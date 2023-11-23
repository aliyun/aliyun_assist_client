//go:build !windows
// +build !windows

package daemon

import (
	"os"
	"syscall"

	"github.com/aliyun/aliyun_assist_client/common/executil"
)

// Daemonize runs this program as daemon.
// Traditional unix's fork-style way would be dangerous to damonize Go program
// since it may break states of underlying runtime scheduler of goroutines. Thus
// simple initiate another process of this program with special setting.
func Daemonize() error {
	executablePath := os.Args[0]
	cmd := executil.Command(executablePath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	cmd.Process.Release()
	os.Exit(0)
	return nil
}
