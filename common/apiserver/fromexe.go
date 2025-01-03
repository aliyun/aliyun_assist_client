package apiserver

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	model "github.com/aliyun/aliyun_assist_client/common/apiserver/fromexemodel"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type ExternalExecutableProvider struct {
	rwLock         sync.RWMutex
	executablePath *string

	pemCerts              *[]byte
	pemCertsError         string
	serverDomain          *string
	serverDomainError     string
	extraHTTPheaders      *map[string]string
	regionId              *string
	regionIdError         string
}

func (p *ExternalExecutableProvider) Name() string {
	p.rwLock.RLock()
	defer p.rwLock.RUnlock()
	if p.executablePath == nil {
		return "<UnknownExternalExecutableProvider>"
	}
	if *p.executablePath == "" {
		return "<NoExternalExecutableProvider>"
	}
	return *p.executablePath
}

func (p *ExternalExecutableProvider) CACertificate(logger logrus.FieldLogger, refresh bool) ([]byte, error) {
	logger = logger.WithField("refresh", refresh)
	if refresh {
		p.rwLock.Lock()
		defer p.rwLock.Unlock()
		p.unsafeProvision(logger)
		return p.unsafeGetCACertificate()
	}

	p.rwLock.RLock()
	if p.executablePath != nil {
		defer p.rwLock.RUnlock()
		return p.unsafeGetCACertificate()
	}

	p.rwLock.RUnlock()
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if p.executablePath == nil {
		p.unsafeProvision(logger)
	}
	return p.unsafeGetCACertificate()
}

func (p *ExternalExecutableProvider) unsafeGetCACertificate() ([]byte, error) {
	if p.executablePath == nil {
		return nil, requester.ErrNotProvided
	}
	if *p.executablePath == "" {
		return nil, requester.ErrNotProvided
	}
	if p.pemCerts == nil {
		return nil, requester.ErrNotProvided
	}
	if p.pemCertsError != "" {
		return nil, errors.New(p.pemCertsError)
	}
	return *p.pemCerts, nil
}

func (p *ExternalExecutableProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	p.rwLock.RLock()
	if p.executablePath != nil {
		defer p.rwLock.RUnlock()
		return p.unsafeGetServerDomain()
	}

	p.rwLock.RUnlock()
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if p.executablePath == nil {
		p.unsafeProvision(logger)
	}
	return p.unsafeGetServerDomain()
}

func (p *ExternalExecutableProvider) unsafeGetServerDomain() (string, error) {
	if p.executablePath == nil {
		return "", requester.ErrNotProvided
	}
	if *p.executablePath == "" {
		return "", requester.ErrNotProvided
	}
	if p.serverDomain == nil {
		return "", requester.ErrNotProvided
	}
	if p.serverDomainError != "" {
		return "", errors.New(p.serverDomainError)
	}
	return *p.serverDomain, nil
}

func (p *ExternalExecutableProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	p.rwLock.RLock()
	if p.executablePath != nil {
		defer p.rwLock.RUnlock()
		return p.unsafeGetExtraHTTPHeaders()
	}

	p.rwLock.RUnlock()
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if p.executablePath == nil {
		p.unsafeProvision(logger)
	}
	return p.unsafeGetExtraHTTPHeaders()
}

func (p *ExternalExecutableProvider) unsafeGetExtraHTTPHeaders() (map[string]string, error) {
	if p.executablePath == nil {
		return nil, requester.ErrNotProvided
	}
	if *p.executablePath == "" {
		return nil, requester.ErrNotProvided
	}
	if p.extraHTTPheaders == nil {
		return nil, requester.ErrNotProvided
	}
	return *p.extraHTTPheaders, nil
}

func (p *ExternalExecutableProvider) RegionId(logger logrus.FieldLogger) (string, error) {
	p.rwLock.RLock()
	if p.executablePath != nil {
		defer p.rwLock.RUnlock()
		return p.unsafeGetRegionId()
	}

	p.rwLock.RUnlock()
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	if p.executablePath == nil {
		p.unsafeProvision(logger)
	}
	return p.unsafeGetRegionId()
}

func (p *ExternalExecutableProvider) unsafeGetRegionId() (string, error) {
	if p.executablePath == nil {
		return "", requester.ErrNotProvided
	}
	if *p.executablePath == "" {
		return "", requester.ErrNotProvided
	}
	if p.regionId == nil {
		return "", requester.ErrNotProvided
	}
	if p.regionIdError != "" {
		return "", errors.New(p.regionIdError)
	}
	return *p.regionId, nil
}

func (p *ExternalExecutableProvider) unsafeProvision(logger logrus.FieldLogger) {
	configDir, err := pathutil.GetConfigPath()
	if err != nil {
		configDir = ""
		logger.WithError(err).Error("Get config path failed")
	}

	crossVersionConfigDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		crossVersionConfigDir = ""
		logger.WithError(err).Error("Get cross version config path failed")
	}
	if configDir == "" && crossVersionConfigDir == "" {
		return
	}

	for _, candidateName := range candidateExternalExecutableProviderNames {
		candidatePath := filepath.Join(configDir, candidateName)
		if _, err := os.Stat(candidatePath); !os.IsNotExist(err) {
			p.executablePath = &candidatePath
			break
		}
	}
	if p.executablePath == nil {
		for _, candidateName := range candidateExternalExecutableProviderNames {
			candidatePath := filepath.Join(crossVersionConfigDir, candidateName)
			if _, err := os.Stat(candidatePath); !os.IsNotExist(err) {
				p.executablePath = &candidatePath
				break
			}
		}
	}
	if p.executablePath == nil {
		noExecutablePath := ""
		p.executablePath = &noExecutablePath
		return
	}

	stdout, stderr, err := runExternalProvider(*p.executablePath)
	logger = logger.WithFields(logrus.Fields{
		"path":   *p.executablePath,
		"stdout": stdout,
		"stderr": stderr,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to run external API server provider")
		return
	}

	var provision model.ProvisionOutputV1
	if err := json.Unmarshal([]byte(stdout), &provision); err != nil {
		logger.WithError(err).Error("Mal-formatted stdout from external provider")
		return
	}
	if provision.SchemaVersion != "1.0" {
		logger.WithError(errors.New("unknown schema version")).Errorf("Failed to parse provider stdout of schema version %s", provision.SchemaVersion)
		return
	}

	var provisionErr model.ErrorOutputV1
	if err := json.Unmarshal([]byte(stderr), &provisionErr); err != nil {
		logger.WithError(err).Error("Mal-formatted stderr from external provider")
	}
	if provisionErr.SchemaVersion != "1.0" {
		logger.WithError(errors.New("unknown schema version")).Errorf("Failed to parse provider stderr of schema version %s", provisionErr.SchemaVersion)
		return
	}

	pemCerts := []byte(provision.Result.CACertificate)
	p.pemCerts = &pemCerts
	p.serverDomain = &provision.Result.ServerDomain
	p.extraHTTPheaders = &provision.Result.ExtraHTTPHeaders
	p.regionId = &provision.Result.RegionId
	for _, errmsg := range provisionErr.Error {
		switch errmsg.Code {
		case model.CodeGetServerDomainError:
			p.serverDomainError = errmsg.Message
		case model.CodeGetCACertificateError:
			p.pemCertsError = errmsg.Message
		case model.CodeRegionIdError:
			p.regionIdError = errmsg.Message
		}
	}
	logger.WithField("provision", p).Info("Provisioned API server information with external provider")
}
