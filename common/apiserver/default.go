package apiserver

import (
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

var (
	_envProvider = &EnvironmentVariableProvider{}
	_externalExecutableProvider = &ExternalExecutableProvider{}
	_generalProvider = &GeneralProvider{}
	_hybridModeProvider = &HybridModeProvider{}
	_inherentCAFileProvider = &InherentCAFileProvider{}
	cachedCAFileProvider = &CachedCAFileProvider{}
	metaserverProvider = &MetaserverProvider{}
	regionidFileProvider = &RegionIdFileProvider{}

	defaultRootCAProviders = []requester.CACertificateProvider{
		_envProvider,
		_externalExecutableProvider,
		cachedCAFileProvider,
		_inherentCAFileProvider,
		metaserverProvider,
	}

	defaultAPIServerProviders = []requester.APIServerProvider{
		_envProvider,
		_externalExecutableProvider,
		_hybridModeProvider,
		metaserverProvider,
		_generalProvider,
	}

	fallbackRegionIdProviders = []requester.RegionIdProvider{
		_externalExecutableProvider,
		_hybridModeProvider,
		metaserverProvider,
		regionidFileProvider,
	}
)

func init() {
	requester.SetRootCAProviders(defaultRootCAProviders)
	requester.SetAPIServerProviders(defaultAPIServerProviders)
	requester.SetRegionIdProviders(fallbackRegionIdProviders)
}
