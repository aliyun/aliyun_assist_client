package channel

import (
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/stretchr/testify/assert"
)

func TestNewSessionChannel(t *testing.T) {
	type args struct {
		url                       string
		sessionId                 string
		inputStreamMessageHandler InputStreamMessageHandler
		cancelFlag                util.CancelFlag
	}
	theArgs := args{
		url:                       "url",
		sessionId:                 "sessionId",
		inputStreamMessageHandler: func(streamDataMessage message.Message) error { return nil },
		cancelFlag:                util.NewChanneledCancelFlag(),
	}
	tests := []struct {
		name    string
		args    args
		want    *SessionChannel
		wantErr bool
	}{
		{
			name:    "normal",
			args:    theArgs,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "normal" {
				var c *WebSocketChannel
				guard := gomonkey.ApplyMethod(
					reflect.TypeOf(c),
					"Initialize", func(c *WebSocketChannel, channelUrl string, onMessageHandler func([]byte), onErrorHandler func(error)) {
						return
					})
				defer guard.Reset()
			}
			NewSessionChannel(tt.args.url, tt.args.sessionId, tt.args.inputStreamMessageHandler, tt.args.cancelFlag)
		})
	}
}

func TestSendKeyExchangeMessage(t *testing.T) {
	sessionChannel := &SessionChannel{
		logger:                   log.GetLogger().WithField("channelId", "testSessionId"),
		StreamDataSequenceNumber: 1234,
	}
	inputData := []byte("test")

	var agentMessage *message.Message
	defer gomonkey.ApplyMethod(reflect.TypeOf(agentMessage), "Serialize", func(_ *message.Message) ([]byte, error) {
		return []byte("mockMessageSerialized"), nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(sessionChannel), "SendMessage", func(_ *SessionChannel, _ []byte, _ int) error {
		return nil
	}).Reset()

	err := sessionChannel.SendKeyExchangeMessage(inputData)
	assert.Nil(t, err)
}
