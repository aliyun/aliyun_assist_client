package channel

import (
	"errors"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func TestNewSessionChannel(t *testing.T) {
	type args struct {
		url                       string
		sessionId                 string
		inputStreamMessageHandler InputStreamMessageHandler
		cancelFlag                util.CancelFlag
	}
	theArgs := args{
		url: "url",
		sessionId: "sessionId",
		inputStreamMessageHandler: func(streamDataMessage message.Message) error { return nil },
		cancelFlag: util.NewChanneledCancelFlag(),

	}
	tests := []struct {
		name    string
		args    args
		want    *SessionChannel
		wantErr bool
	}{
		{
			name: "wsChannelInitializeError",
			args: theArgs,
			wantErr: true,
		},
		{
			name: "normal",
			args: theArgs,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "wsChannelInitializeError" {
				var c *WebSocketChannel
				guard := gomonkey.ApplyMethod(
					reflect.TypeOf(c), 
					"Initialize", func(c *WebSocketChannel, channelUrl string, onMessageHandler func([]byte), onErrorHandler func(error)) error { return errors.New("some errir") })
				defer guard.Reset()
			} else {
				var c *WebSocketChannel
				guard := gomonkey.ApplyMethod(
					reflect.TypeOf(c), 
					"Initialize", func(c *WebSocketChannel, channelUrl string, onMessageHandler func([]byte), onErrorHandler func(error)) error { return nil })
				defer guard.Reset()
			}
			_, err := NewSessionChannel(tt.args.url, tt.args.sessionId, tt.args.inputStreamMessageHandler, tt.args.cancelFlag)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSessionChannel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
