package process

import (
	"context"
	"os/exec"
	"time"
)

// SyncRunSimpleDetached runs command in new session and process group
func SyncRunSimpleDetached(command string, arguments []string, timeout int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout) * time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, arguments...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	// TODO: Configurate Windows-specific attributes to create process in new session and process group
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Setsid: true,
	// 	Setpgid: true,
	// 	Pgid: 0,
	// }

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
