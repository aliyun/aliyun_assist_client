package websocket

import (
	"errors"
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/gorilla/websocket"
)

var g_conn *websocket.Conn = nil
var g_closeEvent chan struct{} = nil
var g_WaitingReply bool = false

func connectWebsocketServer() error {
	if g_conn != nil {
		return nil
	}
	host := util.GetServerHost()
	if host == "" {
		return errors.New("GetServerHost error")
	}
	url := "ws://" + host + "/echo"
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	g_conn = c
	log.GetLogger().Println("connectWebsocketServer ok:", url)
	return nil
}

func DisconnectWebsocketServer() {
	if g_conn == nil {
		return
	}
	//客户端主动断开连接
	log.GetLogger().Println("call DisconnectWebsocketServer")
	err := g_conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.GetLogger().Println("write close:", err)
	}
}

func SendMsgToWebsocketServer(msg string) error {
	err := connectWebsocketServer()
	if err != nil {
		return err
	}
	if !g_WaitingReply {
		go func() {
			g_WaitingReply = true
			for {
				_, message, err := g_conn.ReadMessage()
				if err != nil {
					g_WaitingReply = false
					//客户端调用DisconnectWebsocketServer、服务端断开、网络异常，都会走到这里
					// MyInfo.Println("ReadMessage error, close conn:", err)
					g_conn.Close()
					g_conn = nil
					return
				}
				fmt.Printf("recv: %s", message)
			//	ResponseHandler(string(message), false)
			}
		}()
	}
	return g_conn.WriteMessage(websocket.TextMessage, []byte(msg))
}

func ReplyMsgToWebsocketServer(msg string) error {
	if g_conn == nil {
		return errors.New("websocket conn disconnected")
	}
	return g_conn.WriteMessage(websocket.TextMessage, []byte(msg))
}
