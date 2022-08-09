// +build linux freebsd

package taskengine

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var (
	exitcodePoweroff = 193
	exitcodeReboot   = 194
)

var (
	errnoPhrases = map[syscall.Errno]string{
		syscall.EPERM:        "OperationNotPermitted",          // 0x1
		syscall.ENOENT:       "NoSuchFileOrDirectory",          // 0x2
		syscall.EIO:          "InputOutputError",               // 0x5
		syscall.E2BIG:        "ArgumentListTooLong",            // 0x7
		syscall.ENOEXEC:      "ExecFormatError",                // 0x8
		syscall.EBADF:        "BadFileDescriptor",              // 0x9
		syscall.EAGAIN:       "ResourceTemporarilyUnavailable", // 0xb
		syscall.ENOMEM:       "CannotAllocateMemory",           // 0xc
		syscall.EACCES:       "PermissionDenied",               // 0xd
		syscall.EFAULT:       "BadAddress",                     // 0xe
		syscall.EEXIST:       "FileExists",                     // 0x11
		syscall.ENOTDIR:      "NotADirectory",                  // 0x14
		syscall.EISDIR:       "IsADirectory",                   // 0x15
		syscall.EINVAL:       "InvalidArgument",                // 0x16
		syscall.ENFILE:       "TooManyOpenFilesInSystem",       // 0x17
		syscall.EMFILE:       "TooManyOpenFiles",               // 0x18
		syscall.ETXTBSY:      "TextFileBusy",                   // 0x1a
		syscall.ENOSPC:       "NoEnoughSpace",                  // 0x1c
		syscall.EROFS:        "ReadonlyFileSystem",             // 0x1e
		syscall.EMLINK:       "TooManyLinks",                   // 0x1f
		syscall.ENAMETOOLONG: "FileNameTooLong",                // 0x24
		syscall.ELOOP:        "TooManySymbolicLinkLevels",      // 0x28
		syscall.EDQUOT:       "DiskQuotaExceeded",              // 0x7a
	}
)

func (task *Task) detectHomeDirectory() (string, error) {
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": task.taskInfo.TaskId,
		"Phase":  "Running",
		"Step": "detectHomeDirectory",
	})

	if task.taskInfo.Username != "" {
		specifiedUser, err := user.Lookup(task.taskInfo.Username)
		if err != nil {
			return "", fmt.Errorf("%w: Failed to detect home directory of specified user: %s", ErrHomeDirectoryNotAvailable, err.Error())
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
		if task.envHomeDir == "" {
			return "", fmt.Errorf("%w: Failed to use home directory of specified user as working directory for invocation", ErrDefaultWorkingDirectoryNotAvailable)
		}

		taskLogger.Infof("Detected home directory of specified user %s: %s", task.taskInfo.Username, task.envHomeDir)
		if !util.IsDirectory(task.envHomeDir) {
			return "", fmt.Errorf("%w: Failed to use home directory of specified user as working directory for invocation: %s does not exist", ErrDefaultWorkingDirectoryNotAvailable, task.envHomeDir)
		}

		taskLogger.WithFields(logrus.Fields{
			"workingDirectory": task.envHomeDir,
		}).Infoln("Home directory of specified user is available and used as working directory for invocation")
		return task.envHomeDir, nil
	}

	// 3. When both working directory and user for invocation had not been
	// specified, use home directory of current user running agent instead
	workingDir := task.envHomeDir
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

func (task *Task) categorizeSyscallErrno(err error, prefixDefault presetWrapErrorCode) (presetWrapErrorCode, string) {
	defaultPrefix := presetErrorPrefixes[prefixDefault]

	var errno syscall.Errno
	if errors.As(err, &errno) {
		if errnoPhrase, ok := errnoPhrases[errno]; ok {
			return presetWrapErrorCode(errno), fmt.Sprintf("%s.%s", defaultPrefix, errnoPhrase)
		}
	}

	return prefixDefault, defaultPrefix
}
