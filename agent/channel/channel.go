package channel

import (
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
)

const (
	ChannelNone          = 0
	ChannelGshellType    = 1
	ChannelWebsocketType = 2
)

type OnReceiveMsg func(Msg string, ChannelType int) string

//Abstract base class for channel
type IChannel interface {
	//Is current channel working
	IsWorking() bool
	//Is current channel supported
	IsSupported() bool
	//Get Channel Type
	GetChannelType() int
	//Start channel
	StartChannel() error
	//Stop channel
	StopChannel() error
}

type Channel struct {
	CallBack    OnReceiveMsg
	ChannelType int
	Working     atomicutil.AtomicBoolean
}

func (c *Channel) IsWorking() bool {
	return c.Working.IsSet()
}

func (c *Channel) GetChannelType() int {
	return c.ChannelType
}

func ChannelTypeStr(channelType int) string {
	switch (channelType) {
	case ChannelGshellType:
		return "gshell"
	case ChannelWebsocketType:
		return "websocket"
	default:
		return "unknown"
	}
}
