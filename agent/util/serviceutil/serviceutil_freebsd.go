//go:build freebsd
// +build freebsd

package serviceutil

import (
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

// RestartAgentService restarts Agent service, the Agent service process itself
// should not call this function
func RestartAgentService(logger logrus.FieldLogger) {
	processer := process.ProcessCmd{}
	processer.SyncRunSimple("aliyun-service", []string{"--stop"}, 10)
	processer.SyncRunSimple("aliyun-service", []string{"--start"}, 10)
}
