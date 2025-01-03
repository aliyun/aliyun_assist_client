package apiserver

import (
	"fmt"
	"io"
	"os"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

type EnvironmentVariableProvider struct {
	extraHTTPHeadersProvider atomic.Value
}

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

func (p *EnvironmentVariableProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	domain := os.Getenv("ALIYUN_ASSIST_SERVER_HOST")
	if domain == "" {
		return "", requester.ErrNotProvided
	}

	logger.Info("Get host from env ALIYUN_ASSIST_SERVER_HOST: ", domain)
	if instance.IsHybrid() {
		logger.WithFields(logrus.Fields{
			"ALIYUN_ASSIST_SERVER_HOST": domain,
		}).Info("Would connect to host in hybrid mode due to existed registration")
		p.extraHTTPHeadersProvider.Store(&hybridModeHTTPHeadersProvider)
	}

	return domain, nil
}

func (p *EnvironmentVariableProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	epp := p.extraHTTPHeadersProvider.Load()
	if epp == nil {
		return generalHTTPHeadersProvider.ExtraHTTPHeaders(logger)
	}
	ep, ok := epp.(requester.ExtraHTTPHeadersProvider)
	if !ok {
		return generalHTTPHeadersProvider.ExtraHTTPHeaders(logger)
	}
	return ep.ExtraHTTPHeaders(logger)
}
