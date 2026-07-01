package pluginmodel

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type HealthCheckStrategy string
const (
	HealthCheckByScan HealthCheckStrategy = "byScan"
	HealthCheckByPull HealthCheckStrategy = "byPull"
)

type LocalPlugin interface {
	Name() string

	Version() string

	OSType() string

	Architecture() string
}

type LocalManager interface {
	FindInstalled(logger logrus.FieldLogger) ([]LocalPlugin, error)

	FindUpgradable(logger logrus.FieldLogger) ([]LocalPlugin, error)

	Update(logger logrus.FieldLogger, target RemotePlugin) error

	HealthCheck(logger logrus.FieldLogger, strategy HealthCheckStrategy) ([]PluginStatus, error)
}
