package channel

import (
	"errors"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/gorilla/websocket"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

// IWebSocketChannel is the interface for ControlChannel and DataChannel.
type IWebSocketChannel interface {
	Initialize(channelUrl string,
		onMessageHandler func([]byte),
		onErrorHandler func(error)) error
	Open() error
	Close() error
	StartPings()
	IsActive() bool
	SendMessage(input []byte, inputType int) error
}

type WebSocketChannel struct {
	OnMessage    func([]byte)
	OnError      func(error)
	Connection   *websocket.Conn
	Url          string
	IsOpen       bool
	writeLock    *sync.Mutex
}

func (webSocketChannel *WebSocketChannel) Initialize(channelUrl string,
	onMessageHandler func([]byte),
	onErrorHandler func(error)) error {
	webSocketChannel.Url = channelUrl
	webSocketChannel.OnError = onErrorHandler
	webSocketChannel.OnMessage = onMessageHandler
	return nil
}

func (webSocketChannel *WebSocketChannel) IsActive() bool {
	return webSocketChannel.IsOpen == true
}

func (webSocketChannel *WebSocketChannel) Open() error {

	// initialize the write mutex
	log.GetLogger().Infoln("WebSocketChannel Open")
	webSocketChannel.writeLock = &sync.Mutex{}

	header := http.Header{
		util.UserAgentHeader: []string{util.UserAgentValue},
	}

	ws, err := NewWebsocketUtil(nil).OpenConnection(webSocketChannel.Url, header)
	if err != nil {
		log.GetLogger().Errorln("WebSocketChannel Open failed", err)
		return err
	}

	webSocketChannel.Connection = ws
	webSocketChannel.IsOpen = true
	webSocketChannel.StartPings()

	// spin up a different routine to listen to the incoming traffic
	go func() {

		defer func() {
			if msg := recover(); msg != nil {
				log.GetLogger().Errorf("WebsocketChannel listener run panic: %v", msg)
				log.GetLogger().Errorf("%s: %s", msg, debug.Stack())
			}
		}()

		retryCount := 0
		for {
			if webSocketChannel.IsOpen == false {
				log.GetLogger().Info("Ending the channel listening routine since the channel is closed")
				break
			}

			messageType, rawMessage, err := webSocketChannel.Connection.ReadMessage()

			if err != nil {
				retryCount++
				if retryCount >= 10 {
					log.GetLogger().Warnf("Reach the retry limit for receive messages. Error: %v",  err.Error())
					webSocketChannel.OnError(err)
					break
				}
				log.GetLogger().Debugf(
					"An error happened when receiving the message. Retried times: %d, MessageType: %v, Error: %s",
					retryCount,
					messageType,
					err.Error())

			} else if messageType != websocket.TextMessage && messageType != websocket.BinaryMessage {
				// We only accept text messages which are interpreted as UTF-8 or binary encoded text.
				log.GetLogger().Errorf("Invalid message type %d. We only accept UTF-8 or binary encoded text", messageType)

			} else {
				retryCount = 0
				webSocketChannel.OnMessage(rawMessage)
			}
		}
	}()

	return nil
}

// StartPings starts the pinging process to keep the websocket channel alive.
func (webSocketChannel *WebSocketChannel) StartPings() {

	go func() {
		for {
			if webSocketChannel.IsOpen == false {
				return
			}

			log.GetLogger().Debug("WebsocketChannel: Send ping. Message.")
			webSocketChannel.writeLock.Lock()
			err := webSocketChannel.Connection.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			webSocketChannel.writeLock.Unlock()
			if err != nil {
				webSocketChannel.Close()
				log.GetLogger().Warnf("Error while sending websocket ping: %v", err)
				return
			}
			time.Sleep(60*time.Second)
		}
	}()
}

// Close closes the corresponding connection.
func (webSocketChannel *WebSocketChannel) Close() error {

	log.GetLogger().Info("Closing websocket channel connection to: " + webSocketChannel.Url)
	if webSocketChannel.IsOpen == true {
		// Send signal to stop receiving message
		webSocketChannel.IsOpen = false
		return NewWebsocketUtil( nil).CloseConnection(webSocketChannel.Connection)
	}

	log.GetLogger().Debugf("Websocket channel connection to: " + webSocketChannel.Url + " is already Closed!")
	return nil
}

// SendMessage sends a byte message through the websocket connection.
// Examples of message type are websocket.TextMessage or websocket.Binary
func (webSocketChannel *WebSocketChannel) SendMessage(input []byte, inputType int) error {
	// log.GetLogger().Infoln("SendMessage: ", string(input))
	if webSocketChannel.IsOpen == false {
		return errors.New("Can't send message: Connection is closed.")
	}

	if len(input) < 1 {
		return errors.New("Can't send message: Empty input.")
	}

	webSocketChannel.writeLock.Lock()

	err := webSocketChannel.Connection.WriteMessage(inputType, input)
	webSocketChannel.writeLock.Unlock()
	return err
}