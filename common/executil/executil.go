package executil

import (
	"context"
	"errors"
	"os/exec"
)

// The new version of golang's os/exec library has added the restriction
// "cannot run executable found relative to current directory". In order
// to maintain compatibility, we encapsulated exec.LookPath and exec.Command
// to ignored ErrDot.
func LookPath(file string) (res string, err error) {
	res, err = exec.LookPath(file)
	if errors.Is(err, exec.ErrDot) {
		err = nil
	}
	return
}

func Command(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	return cmd
}

func CommandWithContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil
	}
	return cmd
}