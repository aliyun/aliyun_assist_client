package channel

import (
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/gorilla/websocket"
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	wsChannel := &WebSocketChannel{
	}

	if err := wsChannel.Initialize(
		"ws://127.0.0.1:8090/luban/test",
		nil,
		nil); err != nil {
		log.GetLogger().Errorf("failed to initialize websocket channel for datachannel, error: %s", err)
	}
	if err := wsChannel.Open(); err != nil {
		fmt.Printf("failed to connect data channel with error: %s", err)
	}
	time.Sleep(3*time.Second)
	wsChannel.SendMessage([]byte("123"), websocket.BinaryMessage)
	time.Sleep(30*time.Second)
}