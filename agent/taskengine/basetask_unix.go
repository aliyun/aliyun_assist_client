// +build linux freebsd

package taskengine

import (
	"fmt"
	"os"
	"os/user"

	"github.com/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func (task *Task) detectWorkingDirectory() (string, error) {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": task.taskInfo.TaskId,
		"Phase":  "Running",
		"Step": "detectWorkingDirectory",
	})

	// 1. When working directory for invocation has been specified, just check
	// its existence.
	if task.taskInfo.WorkingDir != "" {
		if !util.IsDirectory(task.taskInfo.WorkingDir) {
			return task.taskInfo.WorkingDir, fmt.Errorf("%w: %s", ErrWorkingDirectoryNotExist, task.taskInfo.WorkingDir)
		}

		taskLogger.WithFields(logrus.Fields{
			"workingDirectory": task.taskInfo.WorkingDir,
		}).Infoln("Specified working directory is available and used for invocation")
		return task.taskInfo.WorkingDir, nil
	}

	// 2. When working directory for invocation had not been specified, use home
	// directory of specified user for invocation instead
	if task.taskInfo.Username != "" {
		specifiedUser, err := user.Lookup(task.taskInfo.Username)
		if err != nil {
			return "", fmt.Errorf("%w: Failed to use home directory of specified user as working directory for invocation: %s", ErrDefaultWorkingDirectoryNotAvailable, err.Error())
		}
		taskLogger.Infof("Detected home directory of specified user %s: %s", specifiedUser.Username, specifiedUser.HomeDir)

		if !util.IsDirectory(specifiedUser.HomeDir) {
			return "", fmt.Errorf("%w: Failed to use home directory of specified user as working directory for invocation: %s does not exist", ErrDefaultWorkingDirectoryNotAvailable, specifiedUser.HomeDir)
		}

		taskLogger.WithFields(logrus.Fields{
			"workingDirectory": specifiedUser.HomeDir,
		}).Infoln("Home directory of specified user is available and used as working directory for invocation")
		return specifiedUser.HomeDir, nil
	}

	// 3. When both working directory and user for invocation had not been
	// specified, use home directory of current user running agent instead
	workingDir := ""
	if userHomeDir, err := os.UserHomeDir(); err == nil {
		log.GetLogger().Infof("Detected HOME environment variable: %s", userHomeDir)
		workingDir = userHomeDir
	} else if currentUser, err := user.Current(); err == nil {
		taskLogger.Infof("Detected home directory of current user %s running agent: %s", currentUser.Username, currentUser.HomeDir)
		workingDir = currentUser.HomeDir
	} else {
		taskLogger.WithError(err).Errorln("Failed to obtain home directory of current user running agent")
	}
	if workingDir != "" {
		if util.IsDirectory(workingDir) {
			taskLogger.WithFields(logrus.Fields{
				"workingDirectory": workingDir,
			}).Infoln("Home directory of current user running agent is available and used as working directory for invocation")
			return workingDir, nil
		} else {
			taskLogger.WithFields(logrus.Fields{
				"candidateWorkingDirectory": workingDir,
			}).Warningln("Home directory of current user running agent does not exist and cannot be used as working directory for invocation")
		}
	}

	// 4. After all, use current working directory of agent as the working
	// directory for invocation at last
	taskLogger.Warningln("Failed to detect working directory and would use working directory of agent by default")
	return "", nil
}
