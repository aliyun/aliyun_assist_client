//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package server

import (
	"net"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

func Listen(logger logrus.FieldLogger, endpoint buses.Endpoint) (net.Listener, error) {
	logger.Info("Start ipc service on socket ", endpoint.String())
	return net.Listen(endpoint.GetProtocol(), endpoint.GetPath())
}
