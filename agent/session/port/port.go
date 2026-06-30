package port

import (
	"context"
	// "errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"

	// "github.com/aliyun/aliyun_assist_client/agent/session/shell"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	sendPackageSize     = 2048                                                   // 发送的payload大小上限，单位 B
	defaultSendSpeed    = 200                                                    // 默认的最大数据发送速率，单位 kbps
	defaultSendInterval = 1000 / (defaultSendSpeed * 1024 / 8 / sendPackageSize) // writeloop的循环间隔时间 单位ms

	waitLocalConnTimeoutSecond = 10
)

type PortPlugin struct {
	id           string
	targetHost   string
	portNumber   int
	dataChannel  channel.ISessionChannel
	conn         net.Conn
	connReady    chan struct{}
	sendInterval int

	exitCtx  context.Context
	exitFunc context.CancelCauseFunc

	logger logrus.FieldLogger
}

func NewPortPlugin(id string, targetHost string, portNumber int, flowLimit int) *PortPlugin {
	if targetHost == "" {
		targetHost = "localhost"
	}
	plugin := &PortPlugin{
		id:           id,
		targetHost:   targetHost,
		portNumber:   portNumber,
		sendInterval: defaultSendInterval,
		logger: log.GetLogger().WithFields(logrus.Fields{
			"sessionType": "portforward",
			"sessionId":   id,
		}),
	}
	plugin.connReady = make(chan struct{})
	if flowLimit > 0 {
		plugin.sendInterval = 1000 / (flowLimit / 8 / sendPackageSize)
	} else {
		flowLimit = defaultSendSpeed * 1024
	}
	plugin.logger.Infof("Init send speed, channelId[%s] speed[%d]bps sendInterval[%d]ms\n", id, flowLimit, plugin.sendInterval)
	return plugin
}

func (p *PortPlugin) Stop() {
	if p.conn == nil {
		return
	}
	err := p.conn.Close()
	p.logger.WithError(err).Info("Close local connection")
}

func (p *PortPlugin) Execute(dataChannel channel.ISessionChannel, cancelFlag util.CancelFlag) (sessionRes *sessionresult.SessionResult) {
	p.dataChannel = dataChannel
	// p.exitCtx and p.exitFunc are initialized in Execute() instead of NewPortPlugin() to ensure
	// that p.exitCtx can be released eventually.
	// Although InputStreamMessageHandler() may be called before Execute(), it will wait until
	// p.conn is successfully created before starting to process data (possibly calling p.exitFunc),
	// so p.exitFunc will not be called before it be initialized.
	p.exitCtx, p.exitFunc = context.WithCancelCause(context.Background())

	defer func() {
		p.logger.Infoln("stop in run PortPlugin")
		p.Stop()

		if err := recover(); err != nil {
			p.logger.Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			sessionRes = sessionresult.NewUnknownError(err)
		}
	}()
	targetHostPort := net.JoinHostPort(p.targetHost, strconv.Itoa(p.portNumber))
	p.logger.Infoln("start port, dial tcp connection to ", targetHostPort)
	var err error
	if p.conn, err = net.DialTimeout("tcp", targetHostPort, time.Second*waitLocalConnTimeoutSecond); err != nil {
		sessionRes = sessionresult.NewOpenTargetPortFailedError(targetHostPort, err)
		p.logger.Errorln(sessionRes)
		return
	}
	close(p.connReady)

	p.logger.Infoln("start port success")
	go func() {
		select {
		case <-cancelFlag.C():
			cancelState := cancelFlag.State()
			p.logger.Info("Cancel flag set to %v in session", cancelState)
			if cancelState == util.ErrerOccurred {
				_, err := cancelFlag.IsErrorOccurred()
				p.logger.WithError(err).Error("An error occurred in session")
				p.exitFunc(err)
			} else {
				p.logger.Info("Session is notified to be ended")
				p.exitFunc(sessionresult.NewNotified())
			}
		case <-p.exitCtx.Done():
			cancelFlag.Set(util.ShutDown)
		}
	}()

	go func() {
		p.writePump()
	}()
	p.logger.Infof("Plugin %s started", p.id)

	<-p.exitCtx.Done()
	contextErr := context.Cause(p.exitCtx)
	var ok bool
	if sessionRes, ok = contextErr.(*sessionresult.SessionResult); !ok {
		sessionRes = sessionresult.NewUnknownError(contextErr)
	}
	p.logger.Infoln("Plugin  done", p.id, sessionRes)

	return
}

func (p *PortPlugin) writePump() {
	defer func() {
		p.logger.Info("writePump done")
		if err := recover(); err != nil {
			p.logger.Infoln("WritePump thread crashed with message ", err)
			fmt.Println("WritePump thread crashed with message: \n", err)
		}
	}()

	packet := make([]byte, sendPackageSize)

	for {
		select {
		case <-p.exitCtx.Done():
			return
		default:
			numBytes, err := p.conn.Read(packet)
			if err != nil {
				p.logger.Infof("Unable to read port: %v", err)
				p.exitFunc(sessionresult.NewReadFromTargetPortFailedError(p.conn.RemoteAddr().String(), err))
				return
			}

			if util.IsVerboseMode() {
				p.logger.Infoln("read data:", string(packet[:numBytes]))
			}

			if err = p.dataChannel.SendStreamDataMessage(packet[:numBytes]); err != nil {
				p.logger.Errorf("Unable to send stream data message: %v", err)
				p.exitFunc(sessionresult.NewSendingDataFailedError(err))
				return
			}

			// Wait for TCP to process more data
			time.Sleep(time.Duration(p.sendInterval) * time.Millisecond)
		}
	}
}

func (p *PortPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	if p.conn == nil {
		p.logger.Infof("InputStreamMessageHandler: connect not ready, wait %d seconds...", waitLocalConnTimeoutSecond)
		timer := time.NewTimer(time.Duration(waitLocalConnTimeoutSecond) * time.Second)
		select {
		case <-p.connReady:
			p.logger.Infoln("InputStreamMessageHandler: connect is ready")
		case <-timer.C:
			p.logger.Infof("InputStreamMessageHandler: connect still not ready after %d seconds", waitLocalConnTimeoutSecond)
			return fmt.Errorf("connection with target host port not ready")
		}
	}

	switch streamDataMessage.MessageType {
	case message.InputStreamDataMessage:
		if _, err := p.conn.Write(streamDataMessage.Payload); err != nil {
			p.logger.Errorf("Unable to write to port, err: %v.", err)
			return err
		}
		if util.IsVerboseMode() {
			p.logger.Infoln("write data:", string(streamDataMessage.Payload))
		}
	case message.StatusDataMessage:
		p.logger.Info("message type: ", streamDataMessage.MessageType)
		if len(streamDataMessage.Payload) > 0 {
			code, err := message.BytesToIntU(streamDataMessage.Payload[0:1])
			p.logger.WithError(err).Info("message status code: ", code)
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
					p.logger.Infof("Set send speed, channelId[%s] speed[%d]bps sendInterval[%d]ms\n", p.id, speed, p.sendInterval)
				case 5:
					p.logger.Info("Exit due to receiving a packet with a close status")
					p.exitFunc(sessionresult.NewOk("a close status packet was received."))
				}
			} else {
				p.logger.Errorf("Parse status code err: %s", err)
			}
		}
	case message.CloseDataChannel:
		p.logger.Info("Exit due to receiving CloseDataChannel packet")
		p.exitFunc(sessionresult.NewOk("a CloseDataChannel packet was received."))
	}
	return nil
}
