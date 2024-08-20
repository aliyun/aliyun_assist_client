package channel

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	_ "github.com/aliyun/aliyun_assist_client/common/apiserver"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	WEBSOCKET_SERVER = "/luban/notify_server"
	MAX_RETRY_COUNT  = 5
)

var (
	wssCoolDownCount = 1  // limit of continuous failed connection
	wssCoolDownTime  = 60 // second
)

type WebSocketChannel struct {
	*Channel
	wskConn                  *websocket.Conn
	lock                     sync.Mutex
	writeLock                sync.Mutex
	consecutiveConnectFailed int
	calmDownUntil            time.Time
}

func (c *WebSocketChannel) IsSupported() bool {
	host := util.GetServerHost()
	if host == "" {
		metrics.GetChannelFailEvent(
			metrics.EVENT_SUBCATEGORY_CHANNEL_WS,
			"errormsg", "websocket channel not supported",
			"type", ChannelTypeStr(c.ChannelType),
		).ReportEvent()
		log.GetLogger().Error("websocket channel not supported")
		return false
	}
	return true
}

func (c *WebSocketChannel) StartChannel() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	errmsg := ""
	defer func() {
		if len(errmsg) > 0 {
			metrics.GetChannelFailEvent(
				metrics.EVENT_SUBCATEGORY_CHANNEL_WS,
				"errmsg", errmsg,
				"type", ChannelTypeStr(c.ChannelType),
			).ReportEvent()
		}
	}()
	if c.consecutiveConnectFailed >= wssCoolDownCount {
		if time.Now().Before(c.calmDownUntil) {
			return errors.New("ws channel is calming down")
		}
		c.consecutiveConnectFailed = 0
		log.GetLogger().Info("ws channel is not calm anymore")
	}
	host := util.GetServerHost()
	if host == "" {
		errmsg = "No available host"
		return errors.New("No available host")
	}

	url := "wss://" + host + WEBSOCKET_SERVER

	logger := log.GetLogger().WithField("url", url)
	header := http.Header{
		requester.UserAgentHeader: []string{requester.UserAgentValue},
	}
	if extraHeaders, err := requester.GetExtraHTTPHeaders(logger); extraHeaders != nil {
		for k, v := range extraHeaders {
			header.Add(k, v)
		}
	} else if err != nil {
		logger.WithError(err).Error("Failed to construct extra HTTP headers")
	}

	var MyDialer = &websocket.Dialer{
		Proxy:            requester.GetProxyFunc(logger),
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs: requester.GetRootCAs(logger),
		},
	}
	var dialErr error
	var conn *websocket.Conn
	conn, _, dialErr = MyDialer.Dial(url, header)
	if dialErr != nil {
		if errors.Is(dialErr, x509.UnknownAuthorityError{}) {
			logger.WithError(dialErr).Error("certificate error, reload certificate and retry")
			certPool := requester.PeekRefreshedRootCAs(logger)
			MyDialer.TLSClientConfig.RootCAs = certPool
			if conn, _, dialErr = MyDialer.Dial(url, header); dialErr != nil {
				errmsg = fmt.Sprintf("dial ws channel errror:%s, url=%s", dialErr.Error(), url)
			} else {
				requester.UpdateRootCAs(logger, certPool)
				logger.Info("certificate updated")
			}
		} else {
			errmsg = fmt.Sprintf("dial ws channel errror:%s, url=%s", dialErr.Error(), url)
		}
	}
	if dialErr != nil {
		c.consecutiveConnectFailed += 1
		if c.consecutiveConnectFailed >= wssCoolDownCount {
			c.calmDownUntil = time.Now().Add(time.Second * time.Duration(wssCoolDownTime))
			errmsg = fmt.Sprintf("dial ws channel errror:%s, url=%s, wss dial failed %d times "+
				"consecutivly, need calm down %d second",
				dialErr.Error(), url, c.consecutiveConnectFailed, wssCoolDownTime)
		} else {
			errmsg = fmt.Sprintf("dial ws channel errror:%s, url=%s", dialErr.Error(), url)
		}
		log.GetLogger().Errorln(dialErr)
		return dialErr
	}
	c.consecutiveConnectFailed = 0
	c.wskConn = conn
	logger.Infoln("Start websocket channel ok! url:", url)
	c.Working.Set()
	c.StartPings(time.Second * 60)
	go func() {
		defer func() {
			if msg := recover(); msg != nil {
				logger.Errorf("WebsocketChannel  run panic: %v", msg)
				logger.Errorf("%s: %s", msg, debug.Stack())
			}
		}()
		retryCount := 0
		for {
			if !c.Working.IsSet() {
				logger.Infoln("websocket channel is closed")
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
					c.Working.Clear()
					logger.Errorf("Reach the retry limit for receive messages. Error: %v", err.Error())
					report := clientreport.ClientReport{
						ReportType: "switch_channel_in_wsk",
						Info:       fmt.Sprintf("start:" + err.Error()),
					}
					clientreport.SendReport(report)
					go c.SwitchChannel()
					break
				}
				logger.Errorf(
					"An error happened when receiving the message. Retried times: %d, MessageType: %v, Error: %s",
					retryCount,
					messageType,
					err.Error())
			} else if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				logger.Errorf("Invalid message type %d. ", messageType)

			} else {
				logger.Infof("wsk recv: %s", string(message))

				content := c.CallBack(string(message), ChannelWebsocketType)
				if content != "" {
					c.writeLock.Lock()
					err := c.wskConn.WriteMessage(websocket.TextMessage, []byte(content))
					c.writeLock.Unlock()
					if err != nil {
						metrics.GetChannelFailEvent(
							metrics.EVENT_SUBCATEGORY_CHANNEL_WS,
							"errormsg", fmt.Sprintf("websocket writing err:%s, content=%s", err.Error(), content),
							"type", ChannelTypeStr(c.ChannelType),
						).ReportEvent()
					}
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
		if err := G_ChannelMgr.SelectAvailableChannelAndReport(ChannelNone, "switch_channel_in_wsk", true); err == nil {
			return nil
		}
		time.Sleep(time.Duration(5) * time.Second)
	}
	metrics.GetChannelSwitchEvent(
		"type", ChannelTypeStr(G_ChannelMgr.GetCurrentChannelType()),
		"reportType", "switch_channel_in_wsk",
		"info", fmt.Sprintf("fail: no available channel"),
	).ReportEvent()

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
	if c.Working.IsSet() {
		c.Working.Clear()
		log.GetLogger().Println("close websocket channel")
		err := c.wskConn.Close()
		if err != nil {
			metrics.GetChannelFailEvent(
				metrics.EVENT_SUBCATEGORY_CHANNEL_WS,
				"errormsg", fmt.Sprintf("close websocket channel error:%s", err),
				"type", ChannelTypeStr(c.ChannelType),
			).ReportEvent()
			log.GetLogger().Println("close websocket channel error:", err)
		}
	}
	return nil
}

func (c *WebSocketChannel) StartPings(pingInterval time.Duration) {

	go func() {
		for {
			if !c.Working.IsSet() {
				return
			}
			log.GetLogger().Infoln("WebsocketChannel: ping...")
			c.writeLock.Lock()
			err := c.wskConn.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			c.writeLock.Unlock()
			if err != nil {
				metrics.GetChannelFailEvent(
					metrics.EVENT_SUBCATEGORY_CHANNEL_WS,
					"errormsg", fmt.Sprintf("Error while sending websocket ping: %s", err.Error()),
					"type", ChannelTypeStr(c.ChannelType),
				).ReportEvent()
				log.GetLogger().Errorf("Error while sending websocket ping: %v", err)
				return
			}
			time.Sleep(pingInterval)
		}
	}()
}

func (c *WebSocketChannel) ResetFailedCount() {
	c.lock.Lock()
	defer c.lock.Unlock()
	log.GetLogger().Info("ws channel: reset fail count")
	c.consecutiveConnectFailed = 0
}

func NewWebsocketChannel(CallBack OnReceiveMsg) *WebSocketChannel {
	w := &WebSocketChannel{
		Channel: &Channel{
			CallBack:    CallBack,
			ChannelType: ChannelWebsocketType,
		},
	}
	w.Working.Clear()
	return w
}
