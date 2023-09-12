package requester

import (
	"sync"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

var (
	_regionIdProviders []RegionIdProvider
	_regionId string
	_regionIdLock sync.Mutex
)

func SetRegionIdProviders(providers []RegionIdProvider) {
	_regionIdLock.Lock()
	defer _regionIdLock.Unlock()

	_regionIdProviders = providers
	_regionId = ""
}

func GetRegionId(logger logrus.FieldLogger) (string, error) {
	_regionIdLock.Lock()
	defer _regionIdLock.Unlock()
	if _regionId != "" {
		return _regionId, nil
	}

	var regionId string
	var err error
	for _, provider := range _regionIdProviders {
		regionId, err = provider.RegionId(logger)
		if err != nil {
			logger.WithError(err).Errorf("Failed to get region ID from %s", provider.Name())
			continue
		}

		_regionId = regionId
		return _regionId, nil
	}
	return "", err
}
