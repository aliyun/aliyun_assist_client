// +build !linux,!darwin,!freebsd

package shm

import (
	"errors"
	"os"
)

var ErrPlatformNotSupported = errors.New(
	"No shared memory support on this platform",
)

func Open(regionName string, flags int, perm os.FileMode) (*os.File, error) {
	return nil, ErrPlatformNotSupported
}

func Unlink(regionName string) error {
	return ErrPlatformNotSupported
}
