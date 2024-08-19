package client

import (
	"context"
	"errors"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

var (
	ErrUnsupportedEndpointProtocol = errors.New("Unsupported endpoint protocol")
)

func ConnectWithTimeout(logger logrus.FieldLogger, endpoint buses.Endpoint, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	return Connect(ctx, logger, endpoint)
}

func Connect(ctx context.Context, logger logrus.FieldLogger, endpoint buses.Endpoint) (*grpc.ClientConn, error) {
	contextDialer, err := getContextDialerForEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(contextDialer),
		grpc.WithBlock(),
	}

	ipcPath := endpoint.GetPath()
	logger.Info("Connect to ", ipcPath)
	conn, err := grpc.DialContext(ctx, ipcPath, opts...)
	if err != nil {
		logger.WithError(err).Error("Failed to dial, will retry")

		if conn, err = grpc.DialContext(ctx, ipcPath, opts...); err != nil {
			logger.WithError(err).Error("Failed to dial again")
			return nil, err
		}
	}

	logger.Info("Dial success.")
	return conn, nil
}
