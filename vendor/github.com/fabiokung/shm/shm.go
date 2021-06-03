// +build darwin freebsd

package shm

import (
	"os"
	"syscall"
	"unsafe"
)

func Open(regionName string, flags int, perm os.FileMode) (*os.File, error) {
	name, err := syscall.BytePtrFromString(regionName)
	if err != nil {
		return nil, err
	}
	fd, _, errno := syscall.Syscall(syscall.SYS_SHM_OPEN,
		uintptr(unsafe.Pointer(name)),
		uintptr(flags), uintptr(perm),
	)
	if errno != 0 {
		return nil, errno
	}
	return os.NewFile(fd, regionName), nil
}

func Unlink(regionName string) error {
	name, err := syscall.BytePtrFromString(regionName)
	if err != nil {
		return err
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_SHM_UNLINK,
		uintptr(unsafe.Pointer(name)), 0, 0,
	); errno != 0 {
		return errno
	}
	return nil
}
