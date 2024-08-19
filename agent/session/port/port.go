package port

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/shell"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	Ok               = "Ok"
	Open_port_failed = "Open_port_failed"
	Read_port_failed = "Read_port_failed"
	IO_socket_error  = "IO_port_failed"
	Unknown_error    = "Unknown_error"
)

const (
	sendPackageSize     = 2048                                                   // 发送的payload大小上限，单位 B
	defaultSendSpeed    = 200                                                    // 默认的最大数据发送速率，单位 kbps
	defaultSendInterval = 1000 / (defaultSendSpeed * 1024 / 8 / sendPackageSize) // writeloop的循环间隔时间 单位ms

	waitLocalConnTimeoutSecond = 10
)

type PortPlugin struct {
	id                 string
	targetHost         string
	portNumber         int
	dataChannel        channel.ISessionChannel
	conn               net.Conn
	connReady          chan struct{}
	reconnectToPort    bool
	reconnectToPortErr chan error
	sendInterval       int

	logger logrus.FieldLogger
}

func NewPortPlugin(id string, targetHost string, portNumber int, flowLimit int) *PortPlugin {
	if targetHost == "" {
		targetHost = "localhost"
	}
	plugin := &PortPlugin{
		id:                 id,
		reconnectToPort:    false,
		targetHost:         targetHost,
		portNumber:         portNumber,
		reconnectToPortErr: make(chan error),
		sendInterval:       defaultSendInterval,
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

	if p.conn.Close() != nil {
		p.conn.Close()
	}
}

func (p *PortPlugin) Execute(dataChannel channel.ISessionChannel, cancelFlag util.CancelFlag) (errorCode string, pluginErr error) {
	p.dataChannel = dataChannel

	defer func() {
		p.logger.Infoln("stop in run PortPlugin")
		p.Stop()

		if err := recover(); err != nil {
			p.logger.Errorf("Error occurred while executing port plugin %s: \n%v", p.id, err)
			// Panic in session port plugin SHOULD NOT disturb the whole agent
			// process
			errorCode = Unknown_error
			if v, ok := err.(error); ok {
				pluginErr = v
			} else {
				pluginErr = fmt.Errorf(fmt.Sprint(err))
			}
		}
	}()
	targetHostPort := net.JoinHostPort(p.targetHost, strconv.Itoa(p.portNumber))
	p.logger.Infoln("start port, dial tcp connection to ", targetHostPort)
	if p.conn, pluginErr = net.DialTimeout("tcp", targetHostPort, time.Second*waitLocalConnTimeoutSecond); pluginErr != nil {
		errorString := fmt.Errorf("Unable to start port: %s", pluginErr)
		p.logger.Errorln(errorString)
		errorCode = Open_port_failed
		return
	}
	close(p.connReady)

	p.logger.Infoln("start port success")
	cancelled := make(chan bool, 1)
	errorCode = Ok
	go func() {
		cancelState := cancelFlag.Wait()
		if cancelFlag.State() == util.Canceled {
			cancelled <- true
			errorCode = shell.Timeout
		}
		if cancelFlag.State() == util.Completed {
			cancelled <- true
			errorCode = shell.Notified
		}
		p.logger.Debugf("Cancel flag set to %v in session", cancelState)
	}()

	done := make(chan string, 1)
	go func() {
		done <- p.writePump()
	}()
	p.logger.Infof("Plugin %s started", p.id)

	select {
	case <-cancelled:
		p.reconnectToPortErr <- errors.New("Session has been cancelled")
		p.logger.Info("The session was cancelled")

	case exitCode := <-done:
		p.logger.Infoln("Plugin  done", p.id, exitCode)
		errorCode = exitCode
	}

	return
}

func (p *PortPlugin) writePump() (errorCode string) {
	defer func() {
		if err := recover(); err != nil {
			p.logger.Infoln("WritePump thread crashed with message ", err)
			fmt.Println("WritePump thread crashed with message: \n", err)
		}
	}()

	packet := make([]byte, sendPackageSize)

	for {
		if p.dataChannel.IsActive() == true {
			numBytes, err := p.conn.Read(packet)
			if err != nil {
				// it may cause goroutines leak, disable retry.
				var exitCode int
				if exitCode = p.onError(err); exitCode == 1 {
					p.logger.Infoln("Reconnection to port is successful, resume reading from port.")
					continue
				}
				p.logger.Infof("Unable to read port: %v", err)
				return Read_port_failed
			}

			if util.IsVerboseMode() {
				p.logger.Infoln("read data:", string(packet[:numBytes]))
			}

			if err = p.dataChannel.SendStreamDataMessage(packet[:numBytes]); err != nil {
				p.logger.Errorf("Unable to send stream data message: %v", err)
				return IO_socket_error
			}
		} else {
			p.logger.Infoln("PortPlugin:writePump stream is closed")
			return IO_socket_error
		}

		// Wait for TCP to process more data
		time.Sleep(time.Duration(p.sendInterval) * time.Millisecond)
	}
}

func (p *PortPlugin) onError(theErr error) int {
	p.logger.Infoln("Encountered reconnect while reading from port: ", theErr)
	p.Stop()
	p.reconnectToPort = true

	p.logger.Debugf("Waiting for reconnection to port!!")
	err := <-p.reconnectToPortErr
	p.logger.Infoln("reconnectToPortErr: ", err)
	if err != nil {
		p.logger.Error(err)
		return 2
	}

	return 1
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
	if p.reconnectToPort {
		p.logger.Infof("InputStreamMessageHandler:Reconnect to %s:%d", p.targetHost, p.portNumber)
		var err error
		p.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", p.targetHost, p.portNumber))
		p.reconnectToPortErr <- err
		if err != nil {
			return err
		}

		p.reconnectToPort = false
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
		if len(streamDataMessage.Payload) > 0 {
			code, err := message.BytesToIntU(streamDataMessage.Payload[0:1])
			if err == nil {
				if code == 7 { // 设置agent的发送速率
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
				}
			} else {
				p.logger.Errorf("Parse status code err: %s", err)
			}
		}
		break
	}
	return nil
}
