package apiserver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type InherentCAFileProvider struct {}

func (*InherentCAFileProvider) Name() string {
	return "InherentCAFileProvider"
}

func (*InherentCAFileProvider) CACertificate(logger logrus.FieldLogger, refresh bool) ([]byte, error) {
	versionedConfigDir, err := pathutil.GetConfigPath()
	if err != nil {
		return nil, err
	}

	certPath := filepath.Join(versionedConfigDir, "GlobalSignRootCA.crt")
	certFile, err := os.Open(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bundled CA certificate file: %w", err)
	}

	pemCerts, err := io.ReadAll(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundled CA certificate file %s: %w", certPath, err)
	}

	return pemCerts, nil
}

type CachedCAFileProvider struct {}

func (*CachedCAFileProvider) Name() string {
	return "CachedCAFileProvider"
}

func (p *CachedCAFileProvider) CACertificate(logger logrus.FieldLogger, refresh bool) ([]byte, error) {
	selfCertPath, err := p.getSelfCertPath()
	if err != nil {
		return nil, err
	}

	selfCertFile, err := os.Open(selfCertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cached CA certificate file: %w", err)
	}

	pemCerts, err := io.ReadAll(selfCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cached CA certificate file %s: %w", selfCertPath, err)
	}

	return pemCerts, nil
}

func (p *CachedCAFileProvider) SaveCACertificate(logger logrus.FieldLogger, pemCerts []byte) {
	selfCertPath, err := p.getSelfCertPath()
	if err != nil {
		logger.WithError(err).Error("Failed to resolve root CA certificate cache path")
	}

	if err := os.WriteFile(selfCertPath, pemCerts, os.FileMode(0o640)); err != nil {
		logger.WithError(err).Error("Failed save root CA certificate to file cache")
	}
}

func (*CachedCAFileProvider) getSelfCertPath() (string, error) {
	crossVersionDir, err := pathutil.GetCrossVersionInboundDir()
	if err != nil {
		return "", err
	}

	selfCertPath := filepath.Join(crossVersionDir, "ca-bundle.crt")
	return selfCertPath, nil
}
