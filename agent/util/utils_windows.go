package util

import (
	"os/exec"
)

func ExeCmdNoWait(cmd string) (error, int) {
	var command *exec.Cmd
	command = exec.Command("sh", "-c", cmd)
	err := command.Start()
	if nil != err {
		return err, 0
	}
	return nil, command.Process.Pid
}
