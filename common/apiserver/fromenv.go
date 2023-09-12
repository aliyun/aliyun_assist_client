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

func (*EnvironmentVariableProvider) CACertificate(logger logrus.FieldLogger) ([]byte, error) {
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

func (*EnvironmentVariableProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	domain := os.Getenv("ALIYUN_ASSIST_SERVER_HOST")
	if domain == "" {
		return "", requester.ErrNotProvided
	}

	return domain, nil
}

func (*EnvironmentVariableProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	return nil, nil
}
