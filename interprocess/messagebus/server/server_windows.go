package server

import (
	"net"

	"github.com/Microsoft/go-winio"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

func Listen(logger logrus.FieldLogger, endpoint buses.Endpoint) (net.Listener, error) {
	pipConfig := &winio.PipeConfig{
		MessageMode: false,
		InputBufferSize: 512,
		OutputBufferSize: 512,
	}
	logger.Info("Start ipc service on named pipe ", endpoint.GetPath())
	return winio.ListenPipe(endpoint.GetPath(), pipConfig)
}
