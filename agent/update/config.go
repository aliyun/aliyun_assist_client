package update

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

const (
	disableUpdateFlagFilename = "disable_update"
	disableBootstrapUpdateFlagFilename = "disable_bootstrap_update"
)

func isUpdatingDisabled() (bool, error) {
	// 1. Detect whether updating is disabled across all installed versions
	crossVersionConfigDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		return false, err
	}
	crossVersionFlagPath := filepath.Join(crossVersionConfigDir, disableUpdateFlagFilename)
	if fileutil.CheckFileIsExist(crossVersionFlagPath) {
		log.GetLogger().Infof("Detected cross-version disabling updating flag %s", crossVersionFlagPath)
		return true, nil
	}

	// 2. Detect whether updating is disabled in this installed version
	currentVersionConfigDir, err := pathutil.GetConfigPath()
	if err != nil {
		return false, err
	}
	currentVersionFlagPath := filepath.Join(currentVersionConfigDir, disableUpdateFlagFilename)
	if fileutil.CheckFileIsExist(currentVersionFlagPath) {
		log.GetLogger().Infof("Detected disabling updating flag of current version %s", currentVersionFlagPath)
		return true, nil
	}

	return false, nil
}

func isBootstrapUpdatingDisabled() (bool, error) {
	// 1. Detect whether bootstrap updating is disabled across all installed versions
	crossVersionConfigDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		return false, err
	}
	crossVersionFlagPath := filepath.Join(crossVersionConfigDir, disableBootstrapUpdateFlagFilename)
	if fileutil.CheckFileIsExist(crossVersionFlagPath) {
		log.GetLogger().Infof("Detected cross-version disabling bootstrap updating flag %s", crossVersionFlagPath)
		return true, nil
	}

	// 2. Detect whether bootstrap updating is disabled in this installed version
	currentVersionConfigDir, err := pathutil.GetConfigPath()
	if err != nil {
		return false, err
	}
	currentVersionFlagPath := filepath.Join(currentVersionConfigDir, disableBootstrapUpdateFlagFilename)
	if fileutil.CheckFileIsExist(currentVersionFlagPath) {
		log.GetLogger().Infof("Detected disabling bootstrap updating flag of current version %s", currentVersionFlagPath)
		return true, nil
	}

	return false, nil
}
