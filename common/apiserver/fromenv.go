package apiserver

import (
	"fmt"
	"io"
	"os"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/common/requester"
)

type EnvironmentVariableProvider struct {}

func (*EnvironmentVariableProvider) Name() string {
	return "EnvironmentVariableProvider"
}

func (*EnvironmentVariableProvider) CACertificate(logger logrus.FieldLogger, refresh bool) ([]byte, error) {
	certPath := os.Getenv("ALIYUN_ASSIST_CERT_PATH")
	if certPath == "" {
		return nil, requester.ErrNotProvided
	}

	certFile, err := os.Open(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CA certificate file configured in ALIYUN_ASSIST_CERT_PATH: %w", err)
	}

	pemCerts, err := io.ReadAll(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate file %s configured in ALIYUN_ASSIST_CERT_PATH: %w", certPath, err)
	}

	return pemCerts, nil
}
