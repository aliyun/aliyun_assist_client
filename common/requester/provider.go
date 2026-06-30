package requester

import (
	"context"
	"errors"
	"net"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type APIServerProvider interface {
	Name() string

	ServerDomain(logger logrus.FieldLogger) (string, error)

	ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error)
}

type ExtraHTTPHeadersProvider interface {
	ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error)
}

type DialContextFuncProvider interface {
	DialContextFunc(logger logrus.FieldLogger, baseDialContext func(ctx context.Context, network, address string) (net.Conn, error)) (func(ctx context.Context, network, address string) (net.Conn, error), error)
}

type CACertificateProvider interface {
	Name() string

	CACertificate(logger logrus.FieldLogger, refresh bool) (pemCerts []byte, err error)
}

type RegionIdProvider interface {
	Name() string

	RegionId(logger logrus.FieldLogger) (string, error)
}

var (
	ErrNotProvided = errors.New("not provided")
)
