// +build linux freebsd

package process

import (
	"os/exec"
	"syscall"
)

func prepareDetachedCommand(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setsid: true,
		Setpgid: true,
		Pgid: 0,
	}

	return nil
}
