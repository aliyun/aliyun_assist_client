// +build linux freebsd

package process

import (
	"context"
	"os/exec"
	"syscall"
	"time"
)

// SyncRunSimpleDetached simply runs command in new session and process group
func SyncRunSimpleDetached(command string, arguments []string, timeout int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout) * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, arguments...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// Setsid: true,
		Setpgid: true,
		Pgid: 0,
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
