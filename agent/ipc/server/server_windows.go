package server

import (
	"net"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/ipc"
	"github.com/Microsoft/go-winio"
)

func listen() (net.Listener, error) {
	pipConfig := &winio.PipeConfig{
		MessageMode: false,
		InputBufferSize: 512,
		OutputBufferSize: 512,
	}
	log.GetLogger().Info("Start ipc service on named pipe ", ipc.NamedPipePath)
	return winio.ListenPipe(ipc.NamedPipePath, pipConfig)
}