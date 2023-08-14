//go:build !windows && !plan9 && !solaris && !aix
// +build !windows,!plan9,!solaris,!aix

package filelock

import (
	"os"
	"syscall"
	"time"
)

// flock acquires an advisory lock on a file descriptor.
func flock(f File, exclusive bool, timeout time.Duration) error {
	var t time.Time
	if timeout != 0 {
		t = time.Now()
	}
	fd := f.Fd()
	flag := syscall.LOCK_NB
	if exclusive {
		flag |= syscall.LOCK_EX
	} else {
		flag |= syscall.LOCK_SH
	}
	for {
		// Attempt to obtain an exclusive lock.
		err := syscall.Flock(int(fd), flag)
		if err == nil {
			return nil
		} else if err != syscall.EWOULDBLOCK {
			return err
		}

		// If we timed out then return an error.
		if timeout != 0 && time.Since(t) > timeout-flockRetryTimeout {
			return os.ErrDeadlineExceeded
		}

		// Wait for a bit and try again.
		time.Sleep(flockRetryTimeout)
	}
}

// funlock releases an advisory lock on a file descriptor.
func funlock(f File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}

// isLockingError checks whether an error object is EWOULDBLOCK under unix-like
// systems.
func isLockingError(err error) bool {
	return err == syscall.EWOULDBLOCK
}

// tryFlock attempts to acquire an advisory lock on a file descriptor.
func tryFlock(f File, exclusive bool) error {
	fd := f.Fd()
	flag := syscall.LOCK_NB
	if exclusive {
		flag |= syscall.LOCK_EX
	} else {
		flag |= syscall.LOCK_SH
	}

	// Attempt to obtain an exclusive lock.
	return syscall.Flock(int(fd), flag)
}
