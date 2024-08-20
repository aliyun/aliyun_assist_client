//go:build !windows
// +build !windows

package processutil

import (
	"os"
	"syscall"
)

// _pidAlive tests whether a process is alive or not by sending it Signal 0,
// since Go otherwise has no way to test this.
func _pidAlive(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err == nil {
		err = proc.Signal(syscall.Signal(0))
	}

	return err == nil
}

// _restartServer tell plugin process restart grpc server 
// by sending it Signal SIGUSR1.
func _restartServer(pid int) error {
	proc, err := os.FindProcess(pid)
	if err == nil {
		return proc.Signal(syscall.SIGUSR1)
	}

	return err
}
