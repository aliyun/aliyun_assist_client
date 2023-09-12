package requester

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type APIServerProvider interface {
	Name() string

	ServerDomain(logger logrus.FieldLogger) (string, error)

	ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error)
}

type CACertificateProvider interface {
	Name() string

	CACertificate(logger logrus.FieldLogger) (pemCerts []byte, err error)
}

type RegionIdProvider interface {
	Name() string

	RegionId(logger logrus.FieldLogger) (string, error)
}

var (
	ErrNotProvided = errors.New("not provided")
)
