package filelock

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// flock acquires an advisory lock on a file descriptor.
func flock(f File, exclusive bool, timeout time.Duration) error {
	var t time.Time
	if timeout != 0 {
		t = time.Now()
	}
	var flags uint32 = windows.LOCKFILE_FAIL_IMMEDIATELY
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}
	for {
		// Fix for https://github.com/etcd-io/bbolt/issues/121. Use byte-range
		// -1..0 as the lock on the database file.
		var m1 uint32 = (1 << 32) - 1 // -1 in a uint32
		err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &windows.Overlapped{
			Offset:     m1,
			OffsetHigh: m1,
		})

		if err == nil {
			return nil
		} else if err != windows.ERROR_LOCK_VIOLATION {
			return err
		}

		// If we timed oumercit then return an error.
		if timeout != 0 && time.Since(t) > timeout-flockRetryTimeout {
			return os.ErrDeadlineExceeded
		}

		// Wait for a bit and try again.
		time.Sleep(flockRetryTimeout)
	}
}

// funlock releases an advisory lock on a file descriptor.
func funlock(f File) error {
	var m1 uint32 = (1 << 32) - 1 // -1 in a uint32
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &windows.Overlapped{
		Offset:     m1,
		OffsetHigh: m1,
	})
}

// isLockingError checks whether an error object is ERROR_LOCK_VIOLATION under
// windows.
func isLockingError(err error) bool {
	return err == windows.ERROR_LOCK_VIOLATION
}

// tryFlock attempts to acquire an advisory lock on a file descriptor.
func tryFlock(f File, exclusive bool) error {
	var flags uint32 = windows.LOCKFILE_FAIL_IMMEDIATELY
	if exclusive {
		flags |= windows.LOCKFILE_EXCLUSIVE_LOCK
	}

	// Fix for https://github.com/etcd-io/bbolt/issues/121. Use byte-range
	// -1..0 as the lock on the database file.
	var m1 uint32 = (1 << 32) - 1 // -1 in a uint32
	return windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, 1, 0, &windows.Overlapped{
		Offset:     m1,
		OffsetHigh: m1,
	})
}
