package channel

import (
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/retry"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/gorilla/websocket"
	"sync/atomic"
	"time"
)

type InputStreamMessageHandler func(streamDataMessage message.Message) error

type ISessionChannel interface {
	Open() error
	Close() error
	Reconnect() error
	SendStreamDataMessage(inputData []byte) (err error)
	GetChannelId() string
	IsActive() bool
}

type SessionChannel struct {
	wsChannel  IWebSocketChannel
	ChannelId string
	StreamDataSequenceNumber int64
	input_stream_cnt uint32
	inputStreamMessageHandler func(streamDataMessage message.Message) error
}

func NewSessionChannel(url string, sessionId string, inputStreamMessageHandler InputStreamMessageHandler, cancelFlag util.CancelFlag) (*SessionChannel, error) {
	sessionChannel := &SessionChannel{}
	sessionChannel.StreamDataSequenceNumber = 0
	sessionChannel.ChannelId = sessionId
	sessionChannel.input_stream_cnt = 0
	sessionChannel.wsChannel = &WebSocketChannel{
	}
	sessionChannel.inputStreamMessageHandler = inputStreamMessageHandler
	streamMessageHandler := func(input []byte) {
		if err := sessionChannel.inputMessageHandler(input); err != nil {
			log.GetLogger().Errorf("Invalid message %s\n", err)
		}
	}

	onErrorHandler := func(err error) {
		callable := func() (channel interface{}, err error) {
			if err = sessionChannel.Reconnect(); err != nil {
				return sessionChannel, err
			}
			return sessionChannel, nil
		}
		retryer := retry.ExponentialRetryer{
			CallableFunc:        callable,
			GeometricRatio:      2.0,
			InitialDelayInMilli: 100,
			MaxDelayInMilli:     2000,
			MaxAttempts:         10,
		}
		if _, err := retryer.Call(); err != nil {
			cancelFlag.Set(util.Canceled)
			log.GetLogger().Error(err)
		}
	}
	if err := sessionChannel.wsChannel.Initialize(
		url,
		streamMessageHandler,
		onErrorHandler); err != nil {
		log.GetLogger().Errorf("failed to initialize websocket channel for datachannel, error: %s", err)
		return nil, err
	}
	go func () {
		for {
			if atomic.LoadUint32(&sessionChannel.input_stream_cnt) > 60*3 {
				cancelFlag.Set(util.Canceled)
				log.GetLogger().Infoln("timeout in sessionChannel")
				break
			}
			time.Sleep(time.Second)
			atomic.AddUint32(&sessionChannel.input_stream_cnt, 1)
		}

	} ()

	return sessionChannel,nil
}

func (sessionChannel *SessionChannel) IsActive() bool {
	if sessionChannel.wsChannel == nil {
		return false
	}
	return sessionChannel.wsChannel.IsActive()
}

func (sessionChannel *SessionChannel) Open() error {
	if err := sessionChannel.wsChannel.Open(); err != nil {
		return fmt.Errorf("failed to connect data channel with error: %s", err)
	}
	return nil
}

func (sessionChannel *SessionChannel) Close() error {
	log.GetLogger().Infof("Closing datachannel with channel Id %s", sessionChannel.ChannelId)
	return sessionChannel.wsChannel.Close()
}

func (sessionChannel *SessionChannel) Reconnect() error {
	log.GetLogger().Debugf("Reconnecting datachannel: %s", sessionChannel.ChannelId)

	if err := sessionChannel.wsChannel.Close(); err != nil {
		log.GetLogger().Debugf("Closing datachannel failed with error: %s", err)
	}

	if err := sessionChannel.Open(); err != nil {
		return fmt.Errorf("failed to reconnect datachannel with error: %s", err)
	}

	// sessionChannel.Pause = false
	log.GetLogger().Debugf("Successfully reconnected to datachannel %s", sessionChannel.ChannelId)
	return nil
}

func (sessionChannel *SessionChannel) SendMessage( input []byte, inputType int) error {
	return sessionChannel.wsChannel.SendMessage(input, inputType)
}

func (sessionChannel *SessionChannel) GetChannelId() string {
	return sessionChannel.ChannelId
}

func (sessionChannel *SessionChannel) inputMessageHandler(rawMessage []byte) error {

	streamDataMessage := &message.Message{}
	if err := streamDataMessage.Deserialize(rawMessage); err != nil {
		log.GetLogger().Errorf("Cannot deserialize raw message, err: %v.", err)
		return err
	}

	if err := streamDataMessage.Validate(); err != nil {
		log.GetLogger().Errorf("Invalid StreamDataMessage, err: %v.", err)
		return err
	}

	if util.IsVerboseMode() {
		log.GetLogger().Infoln("user input: ", string(rawMessage))
		log.GetLogger().Infoln("user input num: ", streamDataMessage.SequenceNumber)
		log.GetLogger().Infoln("user input payload: ", string(streamDataMessage.Payload))
	}

	atomic.StoreUint32(&sessionChannel.input_stream_cnt, 0)


	switch streamDataMessage.MessageType {
	case message.InputStreamDataMessage:
		 return sessionChannel.handleStreamDataMessage( *streamDataMessage, rawMessage)
	case message.SetSizeDataMessage:
		 return sessionChannel.handleStreamDataMessage( *streamDataMessage, rawMessage)
	case message.StatusDataMessage:
		return sessionChannel.handleStreamDataMessage( *streamDataMessage, rawMessage)
	default:
		log.GetLogger().Warnf("Invalid message type received: %d", streamDataMessage.MessageType)
	}

	return nil
}

func (sessionChannel *SessionChannel) handleStreamDataMessage(
	streamDataMessage message.Message,
	rawMessage []byte) (err error) {

	if err = sessionChannel.inputStreamMessageHandler(streamDataMessage); err != nil {
		return err
	}
	return nil
}

// SendStreamDataMessage sends a data message in a form of AgentMessage for streaming.
func (sessionChannel *SessionChannel) SendStreamDataMessage(inputData []byte) (err error) {
	if len(inputData) == 0 {
		log.GetLogger().Debugf("Ignoring empty stream data payload.")
		return nil
	}

	agentMessage := &message.Message{
		MessageType:   message.OutputStreamDataMessage,
		SchemaVersion:  "1.01",
		SessionId:  sessionChannel.ChannelId,
		CreatedDate:    uint64(time.Now().UnixNano() / 1000000),
		SequenceNumber: sessionChannel.StreamDataSequenceNumber,
		PayloadLength:   uint32(len(inputData)),
		Payload:        inputData,
	}
	msg, err := agentMessage.Serialize()

	if util.IsVerboseMode() {
		log.GetLogger().Infoln("output data: ", string(msg))
		log.GetLogger().Infoln("output data num: ", sessionChannel.StreamDataSequenceNumber)
	}

	if err != nil {
		return fmt.Errorf("cannot serialize StreamData message %v", agentMessage)
	}

	if err = sessionChannel.SendMessage(msg, websocket.BinaryMessage); err != nil {
		if util.IsVerboseMode() {
			log.GetLogger().Errorf("Error sending stream data message %v", err)
		}
	}

	sessionChannel.StreamDataSequenceNumber = sessionChannel.StreamDataSequenceNumber + 1
	return nil
}
