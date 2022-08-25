package host

import (
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

var (
	exitcodePoweroff = 3009
	exitcodeReboot   = 3010
)

func (p *HostProcessor) checkCredentials() (bool, error) {
	if err := process.IsUserValid(p.Username, p.WindowsUserPassword); err != nil {
		return false, taskerrors.NewInvalidUsernameOrPasswordError(err, fmt.Sprintf("UsernameOrPasswordInvalid_%s", p.Username))
	}

	return true, nil
}

func (p *HostProcessor) checkHomeDirectory() (string, error) {
	return "", nil
}

func (p *HostProcessor) checkWorkingDirectory() (string, error) {
	workingDir := p.WorkingDirectory

	if workingDir == "" {
		// When working directory for invocation is not specified, working
		// directory of agent would be used by default.
		return workingDir, nil
	}

	if !util.IsDirectory(workingDir) {
		return workingDir, taskerrors.NewWorkingDirectoryNotExistError(workingDir)
	}
	return workingDir, nil
}
