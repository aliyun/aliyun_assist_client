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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, ipcPath, opts...)
	if err != nil {
		// First connection may failed in windows7/windows2008, so retry again
		// https://github.com/microsoft/go-winio/issues/173
		// https://github.com/microsoft/go-winio/issues/183
		logger.WithError(err).Error("Failed to dial, will retry")

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if conn, err = grpc.DialContext(ctx, ipcPath, opts...); err != nil {
			logger.WithError(err).Error("Failed to dial again")
			return nil, err
		}
	}

	logger.Info("Dial success.")
	return conn, nil
}
