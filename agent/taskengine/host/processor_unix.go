//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package host

import (
	"fmt"
	"os"
	"os/user"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var (
	exitcodePoweroff = 193
	exitcodeReboot   = 194
)

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
		var err error
		userHomeDir, err := os.UserHomeDir()
		if err == nil {
			taskLogger.WithFields(logrus.Fields{
				"HOME": userHomeDir,
			}).Infof("Detected HOME environment variable")
			return userHomeDir, nil
		}

		currentUser, err := user.Current()
		if err == nil {
			taskLogger.Infof("Detected home directory of current user %s running agent: %s", currentUser.Username, currentUser.HomeDir)
			return currentUser.HomeDir, nil
		}

		taskLogger.WithError(err).Errorln("Failed to obtain home directory of current user running agent")
		return "", nil
	}
}

func (p *HostProcessor) checkWorkingDirectory() (string, error) {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": p.TaskId,
		"Phase":  "HostProcessor-PreChecking",
		"Step":   "detectWorkingDirectory",
	})

	// 1. When working directory for invocation has been specified, just check
	// its existence.
	if p.WorkingDirectory != "" {
		if !util.IsDirectory(p.WorkingDirectory) {
			return p.WorkingDirectory, taskerrors.NewWorkingDirectoryNotExistError(p.WorkingDirectory)
		}

		taskLogger.WithFields(logrus.Fields{
			"workingDirectory": p.WorkingDirectory,
		}).Infoln("Specified working directory is available and used for invocation")
		return p.WorkingDirectory, nil
	}

	// 2. When working directory for invocation had not been specified, use home
	// directory of specified user for invocation instead
	if p.Username != "" {
		if p.envHomeDir == "" {
			return "", taskerrors.NewDefaultWorkingDirectoryNotAvailableError("Failed to use home directory of specified user as working directory for invocation")
		}

		taskLogger.Infof("Detected home directory of specified user %s: %s", p.Username, p.envHomeDir)
		if !util.IsDirectory(p.envHomeDir) {
			return "", taskerrors.NewDefaultWorkingDirectoryNotAvailableError(fmt.Sprintf("Failed to use home directory of specified user as working directory for invocation: %s does not exist", p.envHomeDir))
		}

		taskLogger.WithFields(logrus.Fields{
			"workingDirectory": p.envHomeDir,
		}).Infoln("Home directory of specified user is available and used as working directory for invocation")
		return p.envHomeDir, nil
	}

	// 3. When both working directory and user for invocation had not been
	// specified, use home directory of current user running agent instead
	workingDir := p.envHomeDir
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
