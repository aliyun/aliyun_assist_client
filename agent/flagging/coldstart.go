package flagging

import (
	"sync"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/ramflag"
)

const (
	coldstartFlagName = "aliyun-assist-agent-coldstarted"
)

var (
	// _isColdstartFlaggedOnStartup represents whether cold-start flag has been
	// set when agent startup.
	// REMEMBER: cold-start flag should be detected on startup only once.
	// Subsequent detection should just re-use above result.
	_isColdstartFlaggedOnStartup *bool = nil

	_coldstartFlagDetectionLock sync.Mutex
)

// IsColdstart checks whether agent is cold-start after booting and establish the flag.
func IsColdstart() (bool, error) {
	// Acquiring lock is neccesary for always up-to-date state
	_coldstartFlagDetectionLock.Lock()
	defer _coldstartFlagDetectionLock.Unlock()

	flaggingLogger := log.GetLogger().WithFields(logrus.Fields{
		"step": "coldstartFlagging",
	})

	// 1. Re-use result if flag has been detected before
	if _isColdstartFlaggedOnStartup != nil {
		result := !(*_isColdstartFlaggedOnStartup)

		flaggingLogger.WithFields(logrus.Fields{
			"detectedResult": result,
		}).Infoln("Re-use IsColdstart result which has been detected before")
		return result, nil
	}

	// 2. Not detected before, check if flagged now
	coldstartFlagged, err := ramflag.IsExist(coldstartFlagName)
	if err != nil {
		flaggingLogger.WithError(err).Errorln("Failed to detect RAM flag state")
		return false, err
	}
	_isColdstartFlaggedOnStartup = &coldstartFlagged

	if *_isColdstartFlaggedOnStartup {
		flaggingLogger.Infoln("RAM flag existed due to previous startup of agent")
		return false, nil
	}

	// 3. Establish cold-start flag in RAM only once at first startup after booting
	if err := ramflag.Create(coldstartFlagName); err != nil {
		flaggingLogger.WithError(err).Errorln("Failed to create RAM flag")
		// NOTE: Failing to create coldstart flag is considered as warm start
		return false, err
	}

	flaggingLogger.Infoln("RAM flag created for the first startup of agent after booting")
	return true, nil
}
