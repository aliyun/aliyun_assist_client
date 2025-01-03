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
	defer func() {
		_rootCAsInited = true
	}()

	AccumulateRootCAs(logger)(func(cp *x509.CertPool) bool {
		_rootCAs = cp
		return false
	})

	return _rootCAs
}

// AccumulateRootCAs gives each root CA certificate provider a try, accumulates
// the provided certficate into the pool, and then calls the passed-in yield
// function to validate the pool.
//
// The yield function passed MUST return true if accumulation need to continue,
// i.e., validation failed. Otherwise false, and the loop would break.
//
// When no root CA certificate provided by any provider, the default root CA
// certificate pool based on the system pool, represented by `nil`, would be
// yielded to give it a try.
func AccumulateRootCAs(logger logrus.FieldLogger) func (yield func (*x509.CertPool) bool) {
	return func(yield func(*x509.CertPool) bool) {
		var certPool *x509.CertPool
		var pemCerts []byte
		var err error
		for _, provider := range _rootCAProviders {
			// In fact, parameter refresh is only valid for
			// ExternalExecutableProvider.CACertificate, other
			// provider.CACertificate always do refresh
			pemCerts, err = provider.CACertificate(logger, true)
			if err != nil {
				logger.WithError(err).Errorf("Failed to get preferred Root CA certificate from %s", provider.Name())
				continue
			}

			logger.Infof("Selected %s for preferred Root CA certificate", provider.Name())
			if certPool == nil {
				certPool, err = x509.SystemCertPool()
				if err != nil {
					logger.WithError(err).Warning("No system CAs can be retrieved. Only provided Root CA certificate is used")
					certPool = x509.NewCertPool()
				} else {
					certPool = certPool.Clone()
				}
			}
			certPool.AppendCertsFromPEM(pemCerts)
			if !yield(certPool) {
				return
			}
		}

		// No preferred Root CA certificate provided, give back the default
		// root CA certificate pool based on system pool and then give up
		if pemCerts == nil {
			logger.Warning("No preferred Root CA certificate is provided. Only system CAs would be certified.")
			yield(nil)
		}
	}
}

func UpdateRootCAs(logger logrus.FieldLogger, certPool *x509.CertPool) {
	_rootCAsLock.Lock()
	defer _rootCAsLock.Unlock()
	_rootCAs = certPool
}
