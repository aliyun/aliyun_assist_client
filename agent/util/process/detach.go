package process

import (
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/aliyun/aliyun_assist_client/common/executil"
)

const (
	// ExitPlaceholder is simply constatn placeholder for exitcode when failed
	ExitPlaceholderFailed = -1
)

var (
	// ErrInvalidState indicates function enters unexpected state and exits. You
	// should never meet such error.
	ErrInvalidState = errors.New("Invalid state encountered")
)

// SyncRunDetached simply runs command in new session and process group
func SyncRunDetached(command string, arguments []string, timeout int) (exitcode int, status int, err error) {
	cmd := executil.Command(command, arguments...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := prepareDetachedCommand(cmd); err != nil {
		wrapErr := fmt.Errorf("Preparing command failed: %w", err)
		return ExitPlaceholderFailed, Fail, wrapErr
	}

	if err := cmd.Start(); err != nil {
		wrapErr := fmt.Errorf("Starting process failed: %w", err)
		return ExitPlaceholderFailed, Fail, wrapErr
	}

	terminated := make(chan error, 1)
	go func() {
		terminated <- cmd.Wait()
	}()

	timer := time.NewTimer(time.Duration(timeout) * time.Second)
	defer timer.Stop()
	// Either normally finished, or killed due to timeout
	status = Success
	for r := 0; r < 2; r++ {
		select {
		case <-timer.C:
			cmd.Process.Kill()
			status = Timeout
		case exitResult := <-terminated:
			timer.Stop()
			close(terminated)

			if exitResult == nil {
				return 0, status, nil
			} else if exitErr, ok := exitResult.(*exec.ExitError); ok {
				return exitErr.ExitCode(), status, exitErr
			} else {
				return ExitPlaceholderFailed, Fail, exitResult
			}
		}
	}

	// Should never return from this line
	return ExitPlaceholderFailed, Fail, ErrInvalidState
}
