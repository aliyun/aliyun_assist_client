//go:build linux
// +build linux

package serviceutil

import (
	"context"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/systemdutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

// RestartAgentService restarts Agent service, the Agent service process itself
// should not call this function
func RestartAgentService(logger logrus.FieldLogger) {
	if systemdutil.IsRunningSystemd() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(5))
		defer cancel()
		if _, err := systemdutil.RestartUnit(ctx, "aliyun.service"); err != nil {
			logger.WithError(err).Error("failed to restart service by systemd api")
			goto RESTARTBYCOMMAND
		}
		return
	}
RESTARTBYCOMMAND:
	logger.Info("try to restart service by command")
	if err := restartServiceByCommand(); err != nil {
		logger.WithError(err).Error("failed to restart service by command")
	}
}

func restartServiceByCommand() error {
	processer := process.ProcessCmd{}
	processer.SyncRunSimple("aliyun-service", []string{"--stop"}, 10)
	return processer.SyncRunSimple("aliyun-service", []string{"--start"}, 10)
}
