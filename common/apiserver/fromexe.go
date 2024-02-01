package apiserver

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type ExternalExecutableProvider struct {
	rwLock         sync.RWMutex
	executablePath *string

	pemCerts         *[]byte
	serverDomain     *string
	extraHTTPheaders *map[string]string
	regionId         *string
}

type ProvisionOutputV1 struct {
	SchemaVersion string `json:"schemaVersion"`
	Result        struct {
		CACertificate    string            `json:"caCertificate"`
		ServerDomain     string            `json:"serverDomain"`
		ExtraHTTPHeaders map[string]string `json:"extraHTTPHeaders"`
		RegionId         string            `json:"regionId"`
	} `json:"result"`
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
	return *p.regionId, nil
}

func (p *ExternalExecutableProvider) unsafeProvision(logger logrus.FieldLogger) {
	crossVersionConfigDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		logger.WithError(err).Error("Get cross version config path failed")
		return
	}

	for _, candidateName := range candidateExternalExecutableProviderNames {
		candidatePath := filepath.Join(crossVersionConfigDir, candidateName)
		if _, err := os.Stat(candidatePath); !os.IsNotExist(err) {
			p.executablePath = &candidatePath
			break
		}
	}
	if p.executablePath == nil {
		noExecutablePath := ""
		p.executablePath = &noExecutablePath
		return
	}

	stdout, stderr, err := runExternalProvider(*p.executablePath)
	logger = logger.WithFields(logrus.Fields{
		"path": *p.executablePath,
		"stdout": stdout,
		"stderr": stderr,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to run external API server provider")
		return
	}

	var provision ProvisionOutputV1
	if err := json.Unmarshal([]byte(stdout), &provision); err != nil {
		logger.WithError(err).Error("Mal-formatted stdout from external provider")
		return
	}
	if provision.SchemaVersion != "1.0" {
		logger.WithError(errors.New("unknown schema version")).Errorf("Failed to parse provider stdout of schema version %s", provision.SchemaVersion)
		return
	}

	pemCerts := []byte(provision.Result.CACertificate)
	p.pemCerts = &pemCerts
	p.serverDomain = &provision.Result.ServerDomain
	p.extraHTTPheaders = &provision.Result.ExtraHTTPHeaders
	p.regionId = &provision.Result.RegionId
	logger.WithField("provision", p).Info("Provisioned API server information with external provider")
}
