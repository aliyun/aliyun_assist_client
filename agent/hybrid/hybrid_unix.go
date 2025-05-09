//go:build !windows
// +build !windows

package hybrid

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"os"
)

// CleanUpRegisterDataAndExit will clean up hybrid directory and stop service self, should be called
// by Agent process itself.
func CleanUpRegisterDataAndExit() {
	logger := log.GetLogger().WithField("action", "CleanUpRegisterData")
	clean_unregister_data(logger, false)
	os.Exit(0)
}
