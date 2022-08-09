package taskengine

import (
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var (
	exitcodePoweroff = 3009
	exitcodeReboot   = 3010
)

func (task *Task) detectHomeDirectory() (string, error) {
	return "", nil
}

func (task *Task) detectWorkingDirectory() (string, error) {
	workingDir := task.taskInfo.WorkingDir

	if workingDir == "" {
		// When working directory for invocation is not specified, working
		// directory of agent would be used by default.
		return workingDir, nil
	}

	if !util.IsDirectory(workingDir) {
		return workingDir, fmt.Errorf("%w: %s", ErrWorkingDirectoryNotExist, workingDir)
	}
	return workingDir, nil
}

func (task *Task) categorizeSyscallErrno(err error, prefixDefault presetWrapErrorCode) (presetWrapErrorCode, string) {
	return prefixDefault, presetErrorPrefixes[prefixDefault]
}
