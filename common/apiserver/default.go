package apiserver

import (
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

var (
	_envProvider = &EnvironmentVariableProvider{}
	_hybridModeProvider = &HybridModeProvider{}
	_generalProvider = &GeneralProvider{}

	defaultRootCAProviders = []requester.CACertificateProvider{
		_envProvider,
		_generalProvider,
	}

	defaultAPIServerProviders = []requester.APIServerProvider{
		_envProvider,
		_hybridModeProvider,
		_generalProvider,
	}

	defaultRegionIdProviders = []requester.RegionIdProvider{
		_hybridModeProvider,
		_generalProvider,
	}
)

func init() {
	requester.SetRootCAProviders(defaultRootCAProviders)
	requester.SetAPIServerProviders(defaultAPIServerProviders)
	requester.SetRegionIdProviders(defaultRegionIdProviders)
}
