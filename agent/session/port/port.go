package port

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/shell"
	"github.com/aliyun/aliyun_assist_client/agent/util"
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
)

type PortPlugin struct {
	id                 string
	targetHost         string
	portNumber         int
	dataChannel        channel.ISessionChannel
	conn               net.Conn
	reconnectToPort    bool
	reconnectToPortErr chan error
	flowLimit          int
	sendInterval       int
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
	}
	if flowLimit > 0 {
		plugin.sendInterval = 1000 / (flowLimit / 8 / sendPackageSize)
	} else {
		flowLimit = defaultSendSpeed * 1024
	}
	log.GetLogger().Infof("Init send speed, channelId[%s] speed[%d]bps sendInterval[%d]ms\n", id, flowLimit, plugin.sendInterval)
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
		log.GetLogger().Infoln("stop in run PortPlugin")
		p.Stop()

		if err := recover(); err != nil {
			log.GetLogger().Errorf("Error occurred while executing port plugin %s: \n%v", p.id, err)
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
	log.GetLogger().Infoln("start port")
	if p.conn, pluginErr = net.Dial("tcp", fmt.Sprintf("%s:%d", p.targetHost, p.portNumber)); pluginErr != nil {
		errorString := fmt.Errorf("Unable to start port: %s", pluginErr)
		log.GetLogger().Errorln(errorString)
		errorCode = Open_port_failed
		return
	}

	log.GetLogger().Infoln("start port success")
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
		log.GetLogger().Debugf("Cancel flag set to %v in session", cancelState)
	}()

	done := make(chan string, 1)
	go func() {
		done <- p.writePump()
	}()
	log.GetLogger().Infof("Plugin %s started", p.id)

	select {
	case <-cancelled:
		p.reconnectToPortErr <- errors.New("Session has been cancelled")
		log.GetLogger().Info("The session was cancelled")

	case exitCode := <-done:
		log.GetLogger().Infoln("Plugin  done", p.id, exitCode)
		errorCode = exitCode
	}

	return
}

func (p *PortPlugin) writePump() (errorCode string) {
	defer func() {
		if err := recover(); err != nil {
			log.GetLogger().Infoln("WritePump thread crashed with message ", err)
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
				if exitCode = p.onError(); exitCode == 1 {
					log.GetLogger().Infoln("Reconnection to port is successful, resume reading from port.")
					continue
				}
				log.GetLogger().Infof("Unable to read port: %v", err)
				return Read_port_failed
			}

			if util.IsVerboseMode() {
				log.GetLogger().Infoln("read data:", string(packet[:numBytes]))
			}

			if err = p.dataChannel.SendStreamDataMessage(packet[:numBytes]); err != nil {
				log.GetLogger().Errorf("Unable to send stream data message: %v", err)
				return IO_socket_error
			}
		} else {
			log.GetLogger().Infoln("PortPlugin:writePump stream is closed")
			return IO_socket_error
		}

		// Wait for TCP to process more data
		time.Sleep(time.Duration(p.sendInterval) * time.Millisecond)
	}
}

func (p *PortPlugin) onError() int {
	log.GetLogger().Infoln("Encountered reconnect while reading from port ")
	p.Stop()
	p.reconnectToPort = true

	log.GetLogger().Debugf("Waiting for reconnection to port!!")
	err := <-p.reconnectToPortErr
	log.GetLogger().Infoln("reconnectToPortErr: ", err)
	if err != nil {
		log.GetLogger().Error(err)
		return 2
	}

	return 1
}

func (p *PortPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	if p.conn == nil {
		log.GetLogger().Infoln("InputStreamMessageHandler: connect not ready")
		return nil
	}
	if p.reconnectToPort {
		log.GetLogger().Infof("InputStreamMessageHandler:Reconnect to %s:%d", p.targetHost, p.portNumber)
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
			log.GetLogger().Errorf("Unable to write to port, err: %v.", err)
			return err
		}
		if util.IsVerboseMode() {
			log.GetLogger().Infoln("write data:", string(streamDataMessage.Payload))
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
						log.GetLogger().Errorf("Invalid flowLimit: %s", err)
						return err
					}
					p.sendInterval = 1000 / (speed / 8 / sendPackageSize)
					log.GetLogger().Infof("Set send speed, channelId[%s] speed[%d]bps sendInterval[%d]ms\n", p.id, speed, p.sendInterval)
				}
			} else {
				log.GetLogger().Errorf("Parse status code err: %s", err)
			}
		}
		break
	}
	return nil
}
