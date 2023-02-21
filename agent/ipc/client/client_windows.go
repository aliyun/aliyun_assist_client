package client

import (
	"net"
	"time"
	"github.com/Microsoft/go-winio"
	"github.com/aliyun/aliyun_assist_client/agent/ipc"
)

var (
	dialer = func(addr string, t time.Duration) (net.Conn, error) {
		return winio.DialPipe(addr, nil)
	}
)

func getIpcPath() string {
	return ipc.NamedPipePath
}
