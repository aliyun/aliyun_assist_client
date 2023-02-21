package client

import (
	"net"
	"path/filepath"
	"time"
	"os"

	"github.com/aliyun/aliyun_assist_client/agent/ipc"
)

var (
	dialer = func(addr string, t time.Duration) (net.Conn, error) {
		return net.Dial(ipc.Protocol, addr)
	}
)

func getIpcPath() string {
	return filepath.Join(os.TempDir(), ipc.SocketName)
}
