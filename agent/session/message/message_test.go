package message

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestMessage_Serialize(t *testing.T) {
	inputData := []byte("hello world")

	agentMessage := &Message{
		MessageType:    OutputStreamDataMessage,
		SchemaVersion:  "1.1",
		CreatedDate:    uint64(time.Now().UnixNano() / 1000000),
		SequenceNumber: 101,
		PayloadLength:   uint32(len(inputData)),
		Payload:        inputData,
	}
	msg, err := agentMessage.Serialize()

	assert.Equal(t, err, nil)

	assert.Contains(t, string(msg), "hello world")

	agentMessage2 := &Message{
	}
	err = agentMessage2.Deserialize(msg)
	assert.Equal(t, err, nil)
	assert.Equal(t, agentMessage2.SequenceNumber, int64(101))
	assert.Equal(t, agentMessage2.SchemaVersion, "1.1")
	assert.Equal(t, agentMessage2.MessageType, uint32(OutputStreamDataMessage))
	assert.Equal(t, string(agentMessage2.Payload), "hello world")
}