package channel

import (
	"errors"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/gorilla/websocket"
	"net/http"
)

// IWebsocketUtil is the interface for the websocketutil.
type IWebsocketUtil interface {
	OpenConnection(url string, requestHeader http.Header) (*websocket.Conn, error)
	CloseConnection(ws *websocket.Conn) error
}

// WebsocketUtil struct provides functionality around creating and maintaining websockets.
type WebsocketUtil struct {
	dialer *websocket.Dialer
}


func NewWebsocketUtil(dialerInput *websocket.Dialer) *WebsocketUtil {
	var websocketUtil *WebsocketUtil

	if dialerInput == nil {
		websocketUtil = &WebsocketUtil{
			dialer: websocket.DefaultDialer,
		}
	} else {
		websocketUtil = &WebsocketUtil{
			dialer: dialerInput,
		}
	}

	return websocketUtil
}

// OpenConnection opens a websocket connection provided an input url and request header.
func (u *WebsocketUtil) OpenConnection(url string, requestHeader http.Header) (*websocket.Conn, error) {

	log.GetLogger().Infof("Opening websocket connection to: %s", url)

	conn, resp, err := u.dialer.Dial(url, requestHeader)
	if err != nil {
		if resp != nil {
			log.GetLogger().Warnf("Failed to dial websocket, status: %s, err: %s", resp.Status, err)
		} else {
			log.GetLogger().Warnf("Failed to dial websocket: %s", err)
		}
		return nil, err
	}

	log.GetLogger().Infof("Successfully opened websocket connection to: %s", url)

	return conn, err
}

// CloseConnection closes a websocket connection given the Conn object as input.
func (u *WebsocketUtil) CloseConnection(ws *websocket.Conn) error {
	if ws == nil {
		return errors.New("websocket conn object is nil")
	}

	log.GetLogger().Debugf("Closing websocket connection to: %s", ws.RemoteAddr())

	err := ws.Close()
	if err != nil {
		log.GetLogger().Warnf("Failed to close websocket: %s", err)
		return err
	}

	log.GetLogger().Infof("Successfully closed websocket connection to: %s", ws.RemoteAddr())

	return nil
}