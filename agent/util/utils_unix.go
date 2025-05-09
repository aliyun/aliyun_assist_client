//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package util

import (
	"bytes"
	"context"
	"os/exec"
	"syscall"

	"github.com/aliyun/aliyun_assist_client/common/executil"
)

func ExeCmdNoWait(cmd string) (error, int) {
	var command *exec.Cmd
	command = executil.Command("sh", "-c", cmd)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := command.Start()
	if nil != err {
		return err, 0
	}
	return nil, command.Process.Pid
}

func ExeCmd(cmd string) (error, string, string) {
	return ExeCmdWithContext(context.Background(), cmd)
}

// Execute a command with timeout limit
func ExeCmdWithContext(ctx context.Context, cmd string) (error, string, string) {
	var outInfo bytes.Buffer
	var errInfo bytes.Buffer
	command := executil.CommandWithContext(ctx, "sh", "-c", cmd)
	command.Stdout = &outInfo
	command.Stderr = &errInfo
	err := command.Run()
	return err, outInfo.String(), errInfo.String()
}
