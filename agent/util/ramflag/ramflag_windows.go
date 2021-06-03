package ramflag

import (
	"errors"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type ATOM uint16

var (
	ErrNameLimitExceeded = errors.New("The RAM flag name is too long")

	_libKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	_libUser32 = windows.NewLazySystemDLL("user32.dll")
	_procGlobalFindAtom = _libKernel32.NewProc("GlobalFindAtomW")
	_procGlobalAddAtom = _libKernel32.NewProc("GlobalAddAtomW")
	_procGlobalDeleteAtom = _libKernel32.NewProc("GlobalDeleteAtom")
)

func init() {
	// According to https://stackoverflow.com/q/3577077, user32.dll must be loaded
	// before calling GlobalFindAtom* functions in kernel32.dll, otherwise error
	// 0x5 (AccessDenied) would be raised.
	_libUser32.Load()
}

func globalFindAtom(lpString string) (ATOM, error) {
	p0, err := windows.UTF16PtrFromString(lpString)
	if err != nil {
		return ATOM(0), err
	}

	r1, _, errno := syscall.Syscall(_procGlobalFindAtom.Addr(), 1, uintptr(unsafe.Pointer(p0)), 0, 0)
	if errno != windows.ERROR_SUCCESS {
		return ATOM(r1), errno
	}
	return ATOM(r1), nil
}

func globalAddAtom(lpString string) (ATOM, error) {
	p0, err := windows.UTF16PtrFromString(lpString)
	if err != nil {
		return ATOM(0), err
	}

	r1, _, errno := syscall.Syscall(_procGlobalAddAtom.Addr(), 1, uintptr(unsafe.Pointer(p0)), 0, 0)
	if errno != windows.ERROR_SUCCESS {
		return ATOM(r1), errno
	}
	return ATOM(r1), nil
}

func globalDeleteAtom(atom ATOM) error { 
	_, _, errno := syscall.Syscall(_procGlobalDeleteAtom.Addr(), 1, uintptr(atom), 0, 0)
	if errno != windows.ERROR_SUCCESS {
		return errno
	}

	return nil
}

func IsExist(name string) (bool, error) {
	if _, err := globalFindAtom(name); err != nil {
		return false, nil
	}
	return true, nil
}

func Create(name string) error {
	if _, err := globalAddAtom(name); err != nil {
		return err
	}

	return nil
}

func Delete(name string) error {
	atom, err := globalFindAtom(name)
	if err != nil {
		return err
	}

	if err := globalDeleteAtom(atom); err != nil {
		return err
	}
	return nil
}
