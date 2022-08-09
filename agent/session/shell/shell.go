package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"os"
	"runtime"
	"time"
	"unicode/utf8"
)


const(
	Ok         =  "Ok"
	Init_channel_failed = "Init_channel_failed"
	Open_channel_failed = "Open_channel_failed"
	Session_id_duplicate = "Session_id_duplicate"
	Process_data_error = "Process_data_error"
	Open_pty_failed = "Open_pty_failed"
	Timeout = "Timeout"
	Notified = "Notified"
	Unknown_error = "Unknown_error"
)

type SizeData struct {
	Cols uint32 `json:"cols"`
	Rows uint32 `json:"rows"`
}

const (
	sendPackageSize = 1024 // 发送的payload大小上限，单位 B
	defaultSendSpeed = 200 // 默认的最大数据发送速率，单位 kbps
	defaultSendInterval = 1000 / (defaultSendSpeed * 1024 / 8 / sendPackageSize) // writeloop的循环间隔时间 单位ms
)

func NewShellPlugin(id string, cmdContent string, username string, passwordName string, flowLimit int) *ShellPlugin {
	plugin := &ShellPlugin{
		id:id,
		cmdContent:cmdContent,
		username:username,
		passwordName:passwordName,
		sendInterval: defaultSendInterval,
	}
	if flowLimit > 0 {
		plugin.sendInterval = 1000 / (flowLimit / 8 / sendPackageSize)
	} else {
		flowLimit = defaultSendSpeed * 1024
	}
	log.GetLogger().Infof("Init send speed, channelId[%s] speed[%d]bps sendInterval[%d]ms\n", id, flowLimit, plugin.sendInterval)
	return plugin
}


func (p *ShellPlugin) Execute(dataChannel channel.ISessionChannel, cancelFlag util.CancelFlag) string {
	p.dataChannel = dataChannel

	defer func() {
		log.GetLogger().Infoln("stop in run ShellPlugin")
		if err := p.stop(); err != nil {
			log.GetLogger().Errorf("Error occurred while closing pty: %v", err)
		}

		if err := recover(); err != nil {
			log.GetLogger().Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			os.Exit(1)
		}
	}()
	log.GetLogger().Infoln("start pty")
	var err error
	err = StartPty(p)
	if err != nil {
		errorString := fmt.Errorf("Unable to start shell: %s", err)
		log.GetLogger().Errorln(errorString)
		return Open_pty_failed
	}
	log.GetLogger().Infoln("start pty success")
	cancelled := make(chan bool, 1)
	errorCode := Ok
	go func() {
		cancelState := cancelFlag.Wait()
		if cancelFlag.State() == util.Canceled{
			cancelled <- true
			errorCode = Timeout
		}
		if cancelFlag.State() == util.Completed {
			cancelled <- true
			errorCode = Notified
		}
		log.GetLogger().Debugf("Cancel flag set to %v in session", cancelState)
	}()


	done := make(chan string, 1)
	go func() {
		done <- p.writePump()
	}()
	log.GetLogger().Infof("Plugin %s started", p.id)


	select {
	case <-cancelled:
		log.GetLogger().Info("The session was cancelled")


	case exitCode := <-done:
		log.GetLogger().Infoln("Plugin  done", p.id, exitCode)
		errorCode = exitCode
	}

	if runtime.GOOS == "linux" {
		p.waitPid()
	}

	return errorCode
}

func (p *ShellPlugin) writePump() (errorCode string) {
	defer func() {
		if err := recover(); err != nil {
			log.GetLogger().Println("WritePump thread crashed with message: \n", err)
		}
	}()

	stdoutBytes := make([]byte, sendPackageSize)
	reader := bufio.NewReader(p.stdout)

	// Wait for all input commands to run.
	time.Sleep(time.Second)

	var unprocessedBuf bytes.Buffer

	for {
		stdoutBytesLen, err := reader.Read(stdoutBytes)

		if err != nil {
			log.GetLogger().Debugf("Failed to read from pty master: %s", err)
			return Ok
		}

		// unprocessedBuf contains incomplete utf8 encoded unicode bytes returned after processing of stdoutBytes
		if unprocessedBuf, err = p.processStdoutData(stdoutBytes, stdoutBytesLen, unprocessedBuf); err != nil {
			log.GetLogger().Errorf("Error processing stdout data, %v", err)
			return Process_data_error
		}
		// Wait for stdout to process more data
		time.Sleep(time.Duration(p.sendInterval) * time.Millisecond)
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