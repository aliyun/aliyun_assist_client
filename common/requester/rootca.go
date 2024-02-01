package requester

import (
	"crypto/x509"
	"sync"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

var (
	_rootCAProviders []CACertificateProvider
	_rootCAs         *x509.CertPool
	_rootCAsInited   bool
	_rootCAsLock     sync.Mutex
)

func SetRootCAProviders(providers []CACertificateProvider) {
	_rootCAsLock.Lock()
	defer _rootCAsLock.Unlock()

	_rootCAProviders = providers
	_rootCAs = nil
	_rootCAsInited = false
}

func GetRootCAs(logger logrus.FieldLogger) *x509.CertPool {
	logger = logger.WithFields(logrus.Fields{
		"action": "GetRootCAs",
	})
	_rootCAsLock.Lock()
	defer _rootCAsLock.Unlock()
	if _rootCAsInited {
		return _rootCAs
	}
	defer func () {
		_rootCAsInited = true
	}()

	var pemCerts []byte
	var err error
	for _, provider := range _rootCAProviders {
		pemCerts, err = provider.CACertificate(logger, false)
		if err != nil {
			logger.WithError(err).Errorf("Failed to get preferred Root CA certificate from %s", provider.Name())
		} else {
			logger.Infof("Selected %s for preferred Root CA certificate", provider.Name())
			break
		}
	}
	if pemCerts == nil {
		logger.Warning("No preferred Root CA certificate is provided. Only system CAs would be certified.")
		_rootCAs = nil
		return nil
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		logger.Warning("No system CAs can be retrieved. Only provided Root CA certificate is used")
		certPool = x509.NewCertPool()
	}
	certPool.AppendCertsFromPEM(pemCerts)

	_rootCAs = certPool
	return _rootCAs
}

// PeekRefreshedRootCAs returns refreshed certs instead cached, and won't modify the certs cache
func PeekRefreshedRootCAs(logger logrus.FieldLogger) *x509.CertPool {
	logger = logger.WithField("action", "PeekRefreshedRootCAs")
	var pemCerts []byte
	var err error
	for _, provider := range _rootCAProviders {
		// In fact, parameter refresh is only valid for ExternalExecutableProvider.CACertificate, 
		// other provider.CACertificate always do refresh
		pemCerts, err = provider.CACertificate(logger, true)
		if err != nil {
			logger.WithError(err).Errorf("Failed to get preferred Root CA certificate from %s", provider.Name())
		} else {
			logger.Infof("Selected %s for preferred Root CA certificate", provider.Name())
			break
		}
	}
	if pemCerts == nil {
		logger.Warning("No preferred Root CA certificate is provided. Only system CAs would be certified.")
		return nil
	}

	certPool, err := x509.SystemCertPool()
	if err != nil {
		logger.Warning("No system CAs can be retrieved. Only provided Root CA certificate is used")
		certPool = x509.NewCertPool()
	} else {
		certPool = certPool.Clone()
	}
	certPool.AppendCertsFromPEM(pemCerts)

	return certPool
}

func UpdateRootCAs(logger logrus.FieldLogger, certPool *x509.CertPool) {
	_rootCAsLock.Lock()
	defer _rootCAsLock.Unlock()
	_rootCAs = certPool
}