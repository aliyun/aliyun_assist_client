package shell

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	Ok                   = "Ok"
	Init_channel_failed  = "Init_channel_failed"
	Open_channel_failed  = "Open_channel_failed"
	Session_id_duplicate = "Session_id_duplicate"
	Process_data_error   = "Process_data_error"
	Open_pty_failed      = "Open_pty_failed"
	Timeout              = "Timeout"
	Notified             = "Notified"
	Unknown_error        = "Unknown_error"
)

type SizeData struct {
	Cols uint32 `json:"cols"`
	Rows uint32 `json:"rows"`
}

const (
	sendPackageSize     = 1024                                                   // 发送的payload大小上限，单位 B
	defaultSendSpeed    = 200                                                    // 默认的最大数据发送速率，单位 kbps
	defaultSendInterval = 1000 / (defaultSendSpeed * 1024 / 8 / sendPackageSize) // writeloop的循环间隔时间 单位ms
)

func NewShellPlugin(id string, cmdContent string, username string, passwordName string, flowLimit int) *ShellPlugin {
	plugin := &ShellPlugin{
		id:           id,
		cmdContent:   cmdContent,
		username:     username,
		passwordName: passwordName,
		sendInterval: defaultSendInterval,
		logger: log.GetLogger().WithFields(logrus.Fields{
			"sessionType": "shell",
			"sessionId":   id,
		}),
	}
	if flowLimit > 0 {
		plugin.sendInterval = 1000 / (flowLimit / 8 / sendPackageSize)
	} else {
		flowLimit = defaultSendSpeed * 1024
	}
	plugin.logger.Infof("Init send speed, speed[%d]bps sendInterval[%d]ms\n", flowLimit, plugin.sendInterval)
	return plugin
}

func (p *ShellPlugin) Execute(dataChannel channel.ISessionChannel, cancelFlag util.CancelFlag) (errorCode string, pluginErr error) {
	p.dataChannel = dataChannel
	// p.exitCtx and p.exitFunc are initialized in Execute() instead of NewPortPlugin() to ensure 
	// that p.exitCtx can be released eventually. 
	// Although InputStreamMessageHandler() may be called before Execute(), it will wait until 
	// p.conn is successfully created before starting to process data (possibly calling p.exitFunc),
	// so p.exitFunc will not be called before it be initialized.
	p.exitCtx, p.exitFunc = context.WithCancelCause(context.Background())
	errorCode = Ok

	defer func() {
		p.logger.Infoln("stop in run ShellPlugin")
		if err := p.stop(); err != nil {
			p.logger.Errorf("Error occurred while closing pty: %v", err)
		}

		if err := recover(); err != nil {
			p.logger.Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			errorCode = Unknown_error
			if v, ok := err.(error); ok {
				pluginErr = v
			} else {
				pluginErr = fmt.Errorf(fmt.Sprint(err))
			}
		}
	}()
	p.logger.Infoln("start pty")
	pluginErr = StartPty(p)
	if pluginErr != nil {
		errorString := fmt.Sprintf("Unable to start shell: %s", pluginErr)
		p.logger.Errorln(errorString)
		errorCode = Open_pty_failed
		return
	}
	p.logger.Infoln("start pty success")
	go func() {
		select {
		case <-cancelFlag.C():
			cancelState := cancelFlag.State()
			if cancelState == util.Canceled {
				p.exitFunc(errors.New(Timeout))
			} else {
				p.exitFunc(errors.New(Notified))
			}
			p.logger.Debugf("Cancel flag set to %v in session", cancelState)
			p.logger.Info("The session was cancelled")
		case <-p.exitCtx.Done():
			cancelFlag.Set(util.ShutDown)
		}
	}()

	go func() {
		p.writePump()
	}()
	p.logger.Infof("Plugin %s started", p.id)

	<-p.exitCtx.Done()
	errorCode = context.Cause(p.exitCtx).Error()
	p.logger.Infoln("Plugin  done", p.id, errorCode)

	if osutil.GetOsType() == osutil.OSLinux || osutil.GetOsType() == osutil.OSFreebsd {
		p.waitPid()
	}

	return
}

func (p *ShellPlugin) writePump() {
	defer func() {
		if err := recover(); err != nil {
			p.logger.Println("WritePump thread crashed with message: \n", err)
		}
	}()

	stdoutBytes := make([]byte, sendPackageSize)
	reader := bufio.NewReader(p.stdout)

	// Wait for all input commands to run.
	time.Sleep(time.Second)

	var unprocessedBuf bytes.Buffer

	for {
		select {
		case <-p.exitCtx.Done():
			return
		default:
			stdoutBytesLen, err := reader.Read(stdoutBytes)
			if err != nil {
				p.logger.Debugf("Failed to read from pty master: %s", err)
				p.exitFunc(errors.New(Ok))
				return
			}

			// unprocessedBuf contains incomplete utf8 encoded unicode bytes returned after processing of stdoutBytes
			if unprocessedBuf, err = p.processStdoutData(stdoutBytes, stdoutBytesLen, unprocessedBuf); err != nil {
				p.logger.Errorf("Error processing stdout data, %v", err)
				p.exitFunc(errors.New(Process_data_error))
				return
			}
			// Wait for stdout to process more data
			time.Sleep(time.Duration(p.sendInterval) * time.Millisecond)
		}
	}
}

// processStdoutData reads utf8 encoded unicode characters from stdoutBytes and sends it over websocket channel.
func (p *ShellPlugin) processStdoutData(
	stdoutBytes []byte,
	stdoutBytesLen int,
	unprocessedBuf bytes.Buffer) (bytes.Buffer, error) {

	// append stdoutBytes to unprocessedBytes and then read rune from appended bytes to send it over websocket channel
	unprocessedBytes := unprocessedBuf.Bytes()
	unprocessedBytes = append(unprocessedBytes[:], stdoutBytes[:stdoutBytesLen]...)
	runeReader := bufio.NewReader(bytes.NewReader(unprocessedBytes))

	var processedBuf bytes.Buffer
	unprocessedBytesLen := len(unprocessedBytes)
	i := 0
	for i < unprocessedBytesLen {
		// read stdout bytes as utf8 encoded unicode character
		stdoutRune, stdoutRuneLen, err := runeReader.ReadRune()
		if err != nil {
			return processedBuf, fmt.Errorf("failed to read rune from reader: %s", err)
		}

		// Invalid utf8 encoded character results into RuneError.
		if stdoutRune == utf8.RuneError {

			// If invalid character is encountered within last 3 bytes of buffer (utf8 takes 1-4 bytes for a unicode character),
			// then break the loop and leave these bytes in unprocessed buffer for them to get processed later with more bytes returned by stdout.
			if unprocessedBytesLen-i < utf8.UTFMax {
				runeReader.UnreadRune()
				break
			}

			// If invalid character is encountered beyond last 3 bytes of buffer, then the character at ith position is invalid utf8 character.
			// Add invalid byte at ith position to processedBuf in such case and return to client to handle display of invalid character.
			processedBuf.Write(unprocessedBytes[i : i+1])
		} else {
			processedBuf.WriteRune(stdoutRune)
		}
		i += stdoutRuneLen
	}

	if p.dataChannel != nil {
		if err := p.dataChannel.SendStreamDataMessage(processedBuf.Bytes()); err != nil {
			return processedBuf, fmt.Errorf("unable to send stream data message: %s", err)
		}
	}

	// log.GetLogger().Println("data output: ", string(processedBuf.Bytes()))
	// return incomplete utf8 encoded unicode bytes to be processed with next batch of stdoutBytes
	unprocessedBuf.Reset()
	if i < unprocessedBytesLen {
		unprocessedBuf.Write(unprocessedBytes[i:unprocessedBytesLen])
	}
	return unprocessedBuf, nil
}

func (p *ShellPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	if p.stdin == nil || p.stdout == nil {
		// This is to handle scenario when cli/console starts sending size data but pty has not been started yet
		// Since packets are rejected, cli/console will resend these packets until pty starts successfully in separate thread
		p.logger.Error("Pty unavailable. Reject incoming message packet")
		return nil
	}

	switch streamDataMessage.MessageType {
	case message.InputStreamDataMessage:
		// log.GetLogger().Traceln("Input message received: ", streamDataMessage.Payload)
		if err := p.onInputStreamData(streamDataMessage.Payload); err != nil {
			return err
		}
	case message.SetSizeDataMessage:
		var size SizeData
		if err := json.Unmarshal(streamDataMessage.Payload, &size); err != nil {
			p.logger.Errorf("Invalid size message: %s", err)
			return err
		}
		// log.GetLogger().Tracef("Resize data received: cols: %d, rows: %d", size.Cols, size.Rows)
		if err := p.SetSize(size.Cols, size.Rows); err != nil {
			p.logger.Errorf("Unable to set pty size: %s", err)
			return err
		}
	case message.StatusDataMessage:
		if len(streamDataMessage.Payload) > 0 {
			code, err := message.BytesToIntU(streamDataMessage.Payload[0:1])
			if err == nil {
				switch code {
				case 7: // 设置agent的发送速率
					speed, err := message.BytesToIntU(streamDataMessage.Payload[1:]) // speed 单位是 bps
					if speed == 0 {
						break
					}
					if err != nil {
						p.logger.Errorf("Invalid flowLimit: %s", err)
						return err
					}
					p.sendInterval = 1000 / (speed / 8 / sendPackageSize)
					p.logger.Infof("Set send speed, speed[%d]bps sendInterval[%d]ms\n", speed, p.sendInterval)
				case 5:
					p.logger.Info("Exit due to receiving a packet with a close status")
					p.exitFunc(errors.New(Ok))
				}
			} else {
				p.logger.Errorf("Parse status code err: %s", err)
			}
		}
	case message.CloseDataChannel:
		p.logger.Info("Exit due to receiving CloseDataChannel packet")
		p.exitFunc(errors.New(Ok))
	}
	return nil
}
