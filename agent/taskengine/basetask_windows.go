package taskengine

import (
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

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
