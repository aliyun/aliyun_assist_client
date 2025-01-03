package apiserver

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/metaserver"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

type MetaserverProvider struct {}

func (*MetaserverProvider) Name() string {
	return "MetaserverProvider"
}

func (*MetaserverProvider) CACertificate(logger logrus.FieldLogger, refresh bool) ([]byte, error) {
	certURL, err := metaserver.GetServerCrt(logger)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(certURL, "http://") {
		return nil, fmt.Errorf("invalid CA certificate URL from metaserver: %s", certURL)
	}

	pemCertsString, err := httpbase.Get(certURL)
	if err != nil {
		return nil, err
	}

	pemCerts := []byte(pemCertsString)
	go cachedCAFileProvider.SaveCACertificate(logger, pemCerts)
	return pemCerts, nil
}

func (p *MetaserverProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	serverURL, err := metaserver.GetServerUrl(logger)
	if err != nil {
		return "", err
	}
	if len(serverURL) == 0 {
		return "", fmt.Errorf("empty server URL from metaserver")
	}
	serverURL = strings.TrimPrefix(serverURL, "https://")
	return serverURL, nil
}

func (*MetaserverProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	return generalHTTPHeadersProvider.ExtraHTTPHeaders(logger)
}

func (p *MetaserverProvider) RegionId(logger logrus.FieldLogger) (string, error) {
	regionId, err := metaserver.GetRegionId(logger)
	if err != nil {
		logger.WithError(err).Error("")
		return "", requester.ErrNotProvided
	}

	return regionId, nil
}
