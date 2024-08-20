package channel

// import (
// 	"context"
// 	"fmt"
// 	"testing"
// 	"time"

// 	"github.com/aliyun/aliyun_assist_client/agent/log"
// 	"github.com/gorilla/websocket"
// 	"github.com/stretchr/testify/assert"

// 	"net/http"
// )

// func TestChannel(t *testing.T) {
// 	s := startserver()
// 	defer s.Shutdown(context.Background())
// 	defer s.Close()
// 	wsChannel := &WebSocketChannel{
// 	}
	
// 	var resp string
// 	onMsg :=  func(b []byte) {
// 		resp = string(b)
// 	}

// 	if err := wsChannel.Initialize(
// 		"ws://localhost:8090/luban/test",
// 		onMsg,
// 		nil); err != nil {
// 		log.GetLogger().Errorf("failed to initialize websocket channel for datachannel, error: %s", err)
// 	}
// 	if err := wsChannel.Open(); err != nil {
// 		fmt.Printf("failed to connect data channel with error: %s", err)
// 	}
// 	err := wsChannel.SendMessage([]byte("123"), websocket.BinaryMessage)
// 	assert.Nil(t, err)
// 	time.Sleep(time.Second)
// 	assert.Equal(t, "123", resp)
// }


// var upgrader = websocket.Upgrader{
// 	CheckOrigin: func(r *http.Request) bool {
// 		return true // 允许来自任何Origin的连接
// 	},
// }

// // echo 将从WebSocket发送过来的消息回传给客户端
// func echo(w http.ResponseWriter, r *http.Request) {
// 	// 将HTTP连接升级到WebSocket协议
// 	conn, err := upgrader.Upgrade(w, r, nil)
// 	if err != nil {
// 		fmt.Println("Upgrade failed:", err)
// 		return
// 	}
// 	defer conn.Close()

// 	// 读取消息
// 	for {
// 		msgType, msg, err := conn.ReadMessage()
// 		if err != nil {
// 			fmt.Println("Read failed:", err)
// 			break
// 		}
// 		fmt.Printf("Received: %s\n", msg)

// 		// 把消息回传给客户端
// 		err = conn.WriteMessage(msgType, msg)
// 		if err != nil {
// 			fmt.Println("Write failed:", err)
// 			break
// 		}
// 	}
// }

// func startserver() *http.Server {
// 	http.HandleFunc("/luban/test", echo) // 设置处理函数

// 	// 启动WebSocket服务器
// 	fmt.Println("WebSocket server starting on :8090...")
// 	server := &http.Server{Addr: "localhost:8090", Handler: nil}
// 	go func() {
// 		if err := server.ListenAndServe(); err != nil {
// 			fmt.Println("websocket server err: ", err)
// 		}
// 	}()
// 	return server
// }