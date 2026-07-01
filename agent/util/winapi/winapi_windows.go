//go:build windows
// +build windows

package winapi

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"syscall"
	"unsafe"
)

var (
	advapi32                      = syscall.NewLazyDLL("advapi32.dll")
	logonProc                     = advapi32.NewProc("LogonUserW")
	impersonateProc               = advapi32.NewProc("ImpersonateLoggedOnUser")
	revertSelfProc                = advapi32.NewProc("RevertToSelf")
	regDisablePredefinedCacheProc = advapi32.NewProc("RegDisablePredefinedCache")
)

const (
	Logon32LogonInteractive = uintptr(2)
	Logon32LogonNetwork     = uintptr(3)

	logon32ProviderDefault = uintptr(0)
)

func Impersonate(logger logrus.FieldLogger, user string, pass string) error {
	token, err := LogonUser(logger, user, pass, Logon32LogonNetwork)
	if err != nil {
		return err
	}
	defer MustCloseHandle(logger, token)

	if rc, _, ec := syscall.SyscallN(impersonateProc.Addr(), uintptr(token)); rc == 0 {
		return syscall.Errno(ec)
	}
	return nil
}

func MustCloseHandle(logger logrus.FieldLogger, handle syscall.Handle) {
	if err := syscall.CloseHandle(handle); err != nil {
		logger.Errorln(err)
	}
}

func LogonUser(logger logrus.FieldLogger, user, pass string, logonType uintptr) (token syscall.Handle, err error) {
	// ".\0" meaning "this computer:
	domain := [2]uint16{uint16('.'), 0}
	//domain,_ := syscall.UTF16FromString("")

	var pu, pp []uint16
	if pu, err = syscall.UTF16FromString(user); err != nil {
		return
	}
	if pp, err = syscall.UTF16FromString(pass); err != nil {
		return
	}

	if rc, _, ec := syscall.SyscallN(logonProc.Addr(),
		uintptr(unsafe.Pointer(&pu[0])),
		uintptr(unsafe.Pointer(&domain[0])),
		uintptr(unsafe.Pointer(&pp[0])),
		logonType,
		logon32ProviderDefault,
		uintptr(unsafe.Pointer(&token))); rc == 0 {
		err = syscall.Errno(ec)
	}
	return
}

// RevertToSelf reverts the impersonation process.
func RevertToSelf() error {
	if rc, _, ec := syscall.SyscallN(revertSelfProc.Addr()); rc == 0 {
		return syscall.Errno(ec)
	}
	return nil
}

// Disables handle caching of the predefined registry handle for HKEY_CURRENT_USER for the current process.
func RegDisablePredefinedCache() error {
	rc, _, _ := syscall.SyscallN(regDisablePredefinedCacheProc.Addr())
	if rc != 0 {
		return syscall.Errno(rc)
	}
	return nil
}
