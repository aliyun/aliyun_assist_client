package host

import (
	"os/user"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

func (p *HostProcessor) checkCredentials() (bool, error) {
	return true, nil
}

func (p *HostProcessor) checkHomeDirectory() (string, error) {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": p.TaskId,
		"Phase":  "HostProcessor-PreChecking",
		"Step":   "detectHomeDirectory",
	})

	if p.Username != "" {
		specifiedUser, err := user.Lookup(p.Username)
		if err != nil {
			return "", taskerrors.NewHomeDirectoryNotAvailableError(err)
		}

		taskLogger.WithFields(logrus.Fields{
			"homeDirectory": specifiedUser.HomeDir,
		}).Infoln("Home directory of specified user is available")
		return specifiedUser.HomeDir, nil
	} else {
		currentUser, err := user.Current()
		if err == nil {
			taskLogger.Infof("Detected home directory of current user %s running agent: %s", currentUser.Username, currentUser.HomeDir)
			return currentUser.HomeDir, nil
		}

		taskLogger.WithError(err).Errorln("Failed to obtain home directory of current user running agent")
		return "", nil
	}
}