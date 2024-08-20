//go:build windows
// +build windows

package osutil

import (
	"context"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/service"
)

// Use the service state change request sent by the service manager to determine
// whether the os is shutting down.
func IsSystemShutdown(ctx context.Context) (bool, error) {
	serviceState := service.StopOrShutdown.Load()
	log.GetLogger().Info("service state: ", serviceState)
	return serviceState == "Shutdown", nil
}
