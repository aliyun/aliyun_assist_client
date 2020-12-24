package util

import (
	"os/exec"
	"syscall"
)

func ExeCmdNoWait(cmd string) (error, int) {
	var command *exec.Cmd
	command = exec.Command("sh", "-c", cmd)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := command.Start()
	if nil != err {
		return err, 0
	}
	return nil, command.Process.Pid
}
