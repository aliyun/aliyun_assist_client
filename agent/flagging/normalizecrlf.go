package flagging

import (
	"path/filepath"

	"go.uber.org/atomic"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

const (
	disableNormalizeCRLFFlagFilename = "disable_normalize_crlf"
)

var (
	_normalizingCRLFDisabled atomic.Bool
)

func IsNormalizingCRLFDisabled() bool {
	return _normalizingCRLFDisabled.Load()
}

func DetectNormalizingCRLFDisabled() (bool, error) {
	// 1. Detect whether CRLF-normalization is disabled across all installed versions
	crossVersionConfigDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		return false, err
	}
	crossVersionFlagPath := filepath.Join(crossVersionConfigDir, disableNormalizeCRLFFlagFilename)
	if util.CheckFileIsExist(crossVersionFlagPath) {
		log.GetLogger().Infof("Detected cross-version disabling CRLF-normalization flag %s", crossVersionFlagPath)
		_normalizingCRLFDisabled.Store(true)
		return true, nil
	}

	// 2. Detect whether CRLF-normalization is disabled in this installed version
	currentVersionConfigDir, err := pathutil.GetConfigPath()
	if err != nil {
		return false, err
	}
	currentVersionFlagPath := filepath.Join(currentVersionConfigDir, disableNormalizeCRLFFlagFilename)
	if util.CheckFileIsExist(currentVersionFlagPath) {
		log.GetLogger().Infof("Detected disabling CRLF-normalization flag of current version %s", currentVersionFlagPath)
		_normalizingCRLFDisabled.Store(true)
		return true, nil
	}

	return false, nil
}
