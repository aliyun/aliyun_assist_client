package shell

import (
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestShellPlugin_Execute(t *testing.T) {
	shellPlugin := NewShellPlugin("", "", "", "", 200*8*1024)
	go func() {
		shellPlugin.Execute(nil, util.NewChanneledCancelFlag())
	}()
	time.Sleep(1*time.Second)
	_,err := shellPlugin.stdin.Write([]byte("pwd\r\n"))
	assert.Equal(t, err, nil)
	_,err = shellPlugin.stdin.Write([]byte("hostname\r\n"))
	assert.Equal(t, err, nil)
	time.Sleep(3*time.Second)
}