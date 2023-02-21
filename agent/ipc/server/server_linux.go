package server

import (
	"net"
	"os"
	"path/filepath"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/ipc"
)

func listen() (net.Listener, error) {
	sockPath := filepath.Join(os.TempDir(), ipc.SocketName)
	if util.CheckFileIsExist(sockPath) {
		os.Remove(sockPath)
	}
	log.GetLogger().Info("Start ipc service on socket ", sockPath)
	return net.Listen(ipc.Protocol, sockPath)
}