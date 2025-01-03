//go:build !linux
// +build !linux

package daemon

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func Daemonize() error {
	return nil
}

func OperateAssistDaemon(logger logrus.FieldLogger, v any) {}
