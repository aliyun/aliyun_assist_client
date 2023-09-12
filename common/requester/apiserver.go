package requester

import (
	"sync"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

var (
	_apiServerProviders []APIServerProvider
	_selectedAPIServerProvider APIServerProvider
	_apiServerProviderLock sync.RWMutex
)

func SetAPIServerProviders(providers []APIServerProvider) {
	_apiServerProviderLock.Lock()
	defer _apiServerProviderLock.Unlock()

	_apiServerProviders = providers
	_selectedAPIServerProvider = nil
}

func GetServerDomain(logger logrus.FieldLogger) (string, error) {
	domain, err := func () (string, error) {
		_apiServerProviderLock.RLock()
		defer _apiServerProviderLock.RUnlock()
		if _selectedAPIServerProvider == nil {
			return "", ErrNotProvided
		}

		domain, err := _selectedAPIServerProvider.ServerDomain(logger)
		if err != nil {
			logger.WithError(err).Warningf("Previously selected API server provider %s does not work for server domain", _selectedAPIServerProvider.Name())
		}
		return domain, err
	}()
	if err == nil {
		return domain, nil
	}

	_apiServerProviderLock.Lock()
	defer _apiServerProviderLock.Unlock()
	if _selectedAPIServerProvider != nil {
		domain, err := _selectedAPIServerProvider.ServerDomain(logger)
		if err == nil {
			return domain, nil
		}

		logger.WithError(err).Warningf("Newly selected API server provider %s does not work for server domain yet", _selectedAPIServerProvider.Name())
	}
	return unsafeSelectProviderForServerDomain(logger)
}

func GetExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	extraHeaders, err := func () (map[string]string, error) {
		_apiServerProviderLock.RLock()
		defer _apiServerProviderLock.RUnlock()
		if _selectedAPIServerProvider == nil {
			return nil, ErrNotProvided
		}

		extraHeaders, err := _selectedAPIServerProvider.ExtraHTTPHeaders(logger)
		if err != nil {
			logger.WithError(err).Warningf("Previously selected API server provider %s does not work for extra HTTP headers", _selectedAPIServerProvider.Name())
		}
		return extraHeaders, err
	}()
	if err == nil {
		return extraHeaders, nil
	}

	_apiServerProviderLock.Lock()
	defer _apiServerProviderLock.Unlock()
	if _selectedAPIServerProvider != nil {
		extraHeaders, err := _selectedAPIServerProvider.ExtraHTTPHeaders(logger)
		if err == nil {
			return extraHeaders, nil
		}

		logger.WithError(err).Warningf("Newly selected API server provider %s does not work for extra HTTP headers yet", _selectedAPIServerProvider.Name())
	}

	_, err = unsafeSelectProviderForServerDomain(logger)
	if err != nil {
		return nil, err
	}
	if _selectedAPIServerProvider != nil {
		return _selectedAPIServerProvider.ExtraHTTPHeaders(logger)
	} else {
		return nil, ErrNotProvided
	}
}

// unsafeSelectProviderForServerDomain MUST be called with protection under
// correct locking
func unsafeSelectProviderForServerDomain(logger logrus.FieldLogger) (string, error) {
	logger.Infoln("Start selection procedure for API server provider")
	for _, provider := range _apiServerProviders {
		domain, err := provider.ServerDomain(logger)
		if err != nil {
			logger.WithError(err).Errorf("Failed to get API server domain from %s", provider.Name())
			continue
		}

		logger.Infof("Selected %s for API server domain and extra HTTP headers needed", provider.Name())
		_selectedAPIServerProvider = provider
		return domain, nil
	}

	logger.Error("No API server provider is selected")
	return "", ErrNotProvided
}
