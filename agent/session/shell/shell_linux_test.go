package shell

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestShellPlugin_Execute(t *testing.T) {
	shellPlugin := NewShellPlugin("", "", "", "")
	go func() {
		shellPlugin.Execute(nil, util.NewChanneledCancelFlag())
	}()
	time.Sleep(1*time.Second)
	_,err := shellPlugin.stdin.Write([]byte("pwd\n"))
	assert.Equal(t, err, nil)
	_,err = shellPlugin.stdin.Write([]byte("ls /var\n"))
	assert.Equal(t, err, nil)
	time.Sleep(3*time.Second)
}