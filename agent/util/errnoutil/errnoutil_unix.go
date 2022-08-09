// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package errnoutil

import (
	"errors"
	"syscall"
)

// IsNoEnoughSpaceError detects "no space left on device" error
func IsNoEnoughSpaceError(err error) bool {
	return errors.Is(err, syscall.ENOSPC)
}
