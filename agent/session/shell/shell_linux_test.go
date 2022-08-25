package shell

// import (
// 	"github.com/stretchr/testify/assert"
// 	"github.com/aliyun/aliyun_assist_client/agent/util"
// 	"testing"
// 	"time"
// )

// func TestShellPlugin_Execute(t *testing.T) {
// 	shellPlugin := NewShellPlugin("", "", "", "", 200*8*1024)
// 	// go func() {
// 	// 	shellPlugin.Execute(nil, util.NewChanneledCancelFlag())
// 	// }()
// 	shellPlugin.Execute(nil, util.NewChanneledCancelFlag())
// 	time.Sleep(1*time.Second)
// 	_,err := shellPlugin.stdin.Write([]byte("pwd\n"))
// 	assert.Equal(t, nil, err)
// 	_,err = shellPlugin.stdin.Write([]byte("ls /var\n"))
// 	assert.Equal(t, nil, err)
// 	time.Sleep(3*time.Second)
// }