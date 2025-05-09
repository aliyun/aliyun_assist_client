package pluginmodel

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type LocalPlugin interface {
	Name() string

	Version() string
}

type LocalManager interface {
	FindUpgradable(logger logrus.FieldLogger) ([]LocalPlugin, error)

	Update(logger logrus.FieldLogger, target RemotePlugin) error
}
