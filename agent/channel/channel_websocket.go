package channel

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
)

const WEBSOCKET_SERVER = "/luban/notify_server"
const MAX_RETRY_COUNT = 5

type WebSocketChannel struct {
	*Channel
	wskConn *websocket.Conn
	lock    sync.Mutex
}

func (c *WebSocketChannel) IsSupported() bool {
	host := util.GetServerHost()
	if host == "" {
		log.GetLogger().Error("websocket channel not supported")
		return false
	}
	return true
}

func (c *WebSocketChannel) StartChannel() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	host := util.GetServerHost()
	if host == "" {
		return errors.New("No available host")
	}

	url := "wss://" + host + WEBSOCKET_SERVER

	header := http.Header{
		util.UserAgentHeader: []string{util.UserAgentValue},
	}

	if util.IsHybrid() || util.IsSelfHosted() {
		u4 := uuid.New()
		str_request_id := u4.String()

		timestamp := timetool.GetAccurateTime()
		str_timestamp := strconv.FormatInt(timestamp, 10)

		var instance_id string
		path := ""
		if util.IsSelfHosted() {
			path, _ = util.GetSelfhostedPath()
		} else {
			path, _ = util.GetHybridPath()
		}

		content, _ := ioutil.ReadFile(path + "/instance-id")
		instance_id = string(content)

		mid, _ := util.GetMachineID()

		input := instance_id + mid + str_timestamp + str_request_id
		pri_key, _ := ioutil.ReadFile(path + "/pri-key")
		output := util.RsaSign(input, string(pri_key))
		log.GetLogger().Infoln(input, output)

		header.Add("x-acs-instance-id", instance_id)
		header.Add("x-acs-timestamp", str_timestamp)
		header.Add("x-acs-request-id", str_request_id)
		header.Add("x-acs-signature", output)
	}

	var MyDialer = &websocket.Dialer{
		Proxy:            util.GetProxyFunc(),
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs: util.CaCertPool,
		},
	}
	conn, _, err := MyDialer.Dial(url, header)
	log.GetLogger().Infoln(url)
	if err != nil {
		log.GetLogger().Errorln(err)
		return err
	}
	c.wskConn = conn
	log.GetLogger().Infoln("Start websocket channel ok! url:", url)
	c.Working = true
	c.StartPings(time.Second * 60)
	go func() {
		defer func() {
			if msg := recover(); msg != nil {
				log.GetLogger().Errorln("WebsocketChannel  run panic: %v", msg)
				log.GetLogger().Errorln("%s: %s", msg, debug.Stack())
			}
		}()
		retryCount := 0
		for {
			if c.Working == false {
				log.GetLogger().Infoln("websocket channel is closed")
				break
			}
			messageType, message, err := c.wskConn.ReadMessage()
			if err != nil {
				time.Sleep(time.Duration(1) * time.Second)
				retryCount++
				if retryCount >= MAX_RETRY_COUNT {
					c.lock.Lock()
					defer c.lock.Unlock()
					c.wskConn.Close()
					c.Working = false
					log.GetLogger().Errorln("Reach the retry limit for receive messages. Error: %v", err.Error())
					report := clientreport.ClientReport{
						ReportType: "switch_channel_in_wsk",
						Info:       fmt.Sprintf("start:" + err.Error()),
					}
					clientreport.SendReport(report)
					go c.SwitchChannel()
					break
				}
				log.GetLogger().Errorln(
					"An error happened when receiving the message. Retried times: %d, MessageType: %v, Error: %s",
					retryCount,
					messageType,
					err.Error())
			} else if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				log.GetLogger().Errorln("Invalid message type %d. ", messageType)

			} else {
				log.GetLogger().Infoln("wsk recv: %s", string(message))

				content := c.CallBack(string(message), ChannelWebsocketType)
				if content != "" {
					c.wskConn.WriteMessage(websocket.TextMessage, []byte(content))
				}

				retryCount = 0
			}
		}
	}()
	return nil
}

func (c *WebSocketChannel) SwitchChannel() error {
	time.Sleep(time.Duration(1) * time.Second)
	for i := 0; i < 5; i++ {
		if G_ChannelMgr.SelectAvailableChannel(ChannelNone) == nil {
			report := clientreport.ClientReport{
				ReportType: "switch_channel_in_wsk",
				Info:       fmt.Sprintf("success: Current channel is %d", G_ChannelMgr.GetCurrentChannelType()),
			}
			clientreport.SendReport(report)
			return nil
		}
		time.Sleep(time.Duration(5) * time.Second)
	}
	report := clientreport.ClientReport{
		ReportType: "switch_channel_in_wsk",
		Info:       fmt.Sprintf("fail: no available channel"),
	}
	clientreport.SendReport(report)
	return errors.New("no available channel")
}

func (c *WebSocketChannel) StopChannel() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.Working == true {
		c.Working = false
		log.GetLogger().Println("close websocket channel")
		err := c.wskConn.Close()
		if err != nil {
			log.GetLogger().Println("close websocket channel error:", err)
		}
	}
	return nil
}

func (c *WebSocketChannel) StartPings(pingInterval time.Duration) {

	go func() {
		for {
			if c.Working == false {
				return
			}
			log.GetLogger().Infoln("WebsocketChannel: ping...")
			c.lock.Lock()
			err := c.wskConn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			c.lock.Unlock()
			if err != nil {
				log.GetLogger().Errorln("Error while sending websocket ping: %v", err)
				return
			}
			time.Sleep(pingInterval)
		}
	}()
}

func NewWebsocketChannel(CallBack OnReceiveMsg) IChannel {
	return &WebSocketChannel{
		Channel: &Channel{
			CallBack:    CallBack,
			ChannelType: ChannelWebsocketType,
			Working:     false,
		},
	}
}
