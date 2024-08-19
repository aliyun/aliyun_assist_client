package server

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

type RegisterFunc func(sr grpc.ServiceRegistrar)

func ListenAndServe(logger logrus.FieldLogger, endpoint buses.Endpoint, serveErr chan error, registers []RegisterFunc, opt ...grpc.ServerOption) (*grpc.Server, error) {
	listener, err := Listen(logger, endpoint)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"endpoint": endpoint,
		}).WithError(err).Error("Failed to listen on specified endpoint")
		return nil, err
	}

	grpcServer := grpc.NewServer(opt...)
	for _, register := range registers {
		register(grpcServer)
	}

	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.WithFields(logrus.Fields{
				"endpoint": endpoint,
			}).WithError(err).Error("Failed to start service on listening endpoint")
			if serveErr != nil {
				serveErr <- err
			}
		}
	}()
	return grpcServer, nil
}
