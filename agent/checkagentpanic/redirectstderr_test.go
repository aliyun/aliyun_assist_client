package checkagentpanic

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchPanicInfoFromFile(t *testing.T) {
	mokeStderrFile := "mokeStderrFile"
	stderrContentPre := `Agent check point
other logs`
	stderrPanicLine := `panic: runtime error: invalid memory address or nil pointer dereference`
	stderrContentPost := `goroutine 34 [running]:
runtime/debug.Stack()
        /root/.g/versions/1.20.5/src/runtime/debug/stack.go:24 +0x64
main.(*program).run.func1()
        /root/go/aliyun_assist_client/rootcmd.go:301 +0x34
panic({0xa3e120, 0x14f9ae0})
        /root/.g/versions/1.20.5/src/runtime/panic.go:884 +0x1f4
github.com/aliyun/aliyun_assist_client/agent/pluginmanager.InitPluginCheckTimer()
        /root/go/aliyun_assist_client/agent/pluginmanager/pluginmanager.go:64 +0x10c
main.(*program).run(0x0?)
        /root/go/aliyun_assist_client/rootcmd.go:345 +0x8a4
created by main.(*program).Start
        /root/go/aliyun_assist_client/rootcmd.go:254 +0x5c`

	tests := []struct{
		panicLimitSize int
	}{
		{
			panicLimitSize: 100,
		},
		{
			panicLimitSize: 500,
		},
		{
			panicLimitSize: 50 * 1024,
		},
	}
	mokeStderrF, err := os.OpenFile(mokeStderrFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	assert.Equal(t, nil, err)
	mokeStderrF.WriteString(stderrContentPre+"\n")
	mokeStderrF.WriteString(stderrPanicLine+"\n")
	mokeStderrF.WriteString(stderrContentPost+"\n")
	mokeStderrF.Close()
	// defer os.Remove(mokeStderrFile)

	for _, tt := range tests {
		panicInfoSizeLimit = tt.panicLimitSize
		panicInfo := searchPanicInfoFromFile(mokeStderrFile)
		assert.Equal(t, true, len(panicInfo) <= tt.panicLimitSize)
		assert.Equal(t, true, strings.HasPrefix(panicInfo, "panic"))
		// fmt.Println(len(panicInfo), tt.panicLimitSize)
	}
}
