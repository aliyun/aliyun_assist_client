//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package acspluginmanager

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/term"

	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

func prepareCmdOptions(executeParams *ExecuteParams) ([]process.CmdOption, error) {
	var options []process.CmdOption
	if executeParams.Foreground {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			options = append(options, optInForeground)
		}
	}

	return options, nil
}

func optInForeground(c *exec.Cmd) error {
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}

	c.SysProcAttr.Foreground = true
	c.SysProcAttr.Ctty = int(os.Stdin.Fd())
	return nil
}
