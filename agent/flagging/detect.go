package flagging

import (
	"path/filepath"

	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

func detectFlagSet(flagName, flagFileName string, theFlag *atomic.Bool) (bool, error) {
	// 1. Detect whether flag is set across all installed versions
	crossVersionConfigDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		return false, err
	}
	crossVersionFlagPath := filepath.Join(crossVersionConfigDir, flagFileName)
	if fileutil.CheckFileIsExist(crossVersionFlagPath) {
		log.GetLogger().Infof("Detected cross-version flag[%s] %s", flagName,  crossVersionFlagPath)
		theFlag.Store(true)
		return true, nil
	}

	// 2. Detect whether flag is set in this installed version
	currentVersionConfigDir, err := pathutil.GetConfigPath()
	if err != nil {
		return false, err
	}
	currentVersionFlagPath := filepath.Join(currentVersionConfigDir, flagFileName)
	if fileutil.CheckFileIsExist(currentVersionFlagPath) {
		log.GetLogger().Infof("Detected flag[%s] of current version %s", flagName, currentVersionFlagPath)
		theFlag.Store(true)
		return true, nil
	}

	return false, nil
}