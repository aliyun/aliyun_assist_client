package process

import (
	"errors"
	"syscall"
	"unsafe"
	"os"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"golang.org/x/sys/windows"
)

var (
	advapi32          = syscall.NewLazyDLL("advapi32.dll")
	logonProc         = advapi32.NewProc("LogonUserW")
	impersonateProc   = advapi32.NewProc("ImpersonateLoggedOnUser")
	revertSelfProc    = advapi32.NewProc("RevertToSelf")
)

const (
	//logon32LogonNetwork          = uintptr(3)
	logon32LOGONINTERACTIVE    = uintptr(2)
	logon32ProviderDefault       = uintptr(0)
)

func (p *ProcessCmd) prepareProcess() error {
	// if p.command.SysProcAttr == nil {
	// 	p.command.SysProcAttr = &windows.SysProcAttr{}
	// }
	// 1. Duplicate current environment variable settings as base
	var env []string
	if p.command.Env == nil || len(p.command.Env) == 0 {
		env = os.Environ()
	} else {
		// append specific envs to osEnv, the value of repetitive key will be covered
		env = os.Environ()
		for i:=0; i<len(p.command.Env); i++ {
			env = append(env, p.command.Env[i])
		}
	}
	p.command.Env = env
	return nil
}

func (p *ProcessCmd)  addCredential () error {
	log.GetLogger().Infoln("addCredential")

	vm_password,err := util.GetSecretParam(p.password)
	if err != nil {
		log.GetLogger().Errorln("get password failed", err)
		return err
	}

	token, err := logonUser(p.user_name, vm_password)
	if err != nil {
		return err
	}

	p.command.SysProcAttr = &windows.SysProcAttr{
		Token: syscall.Token(token),
	}

	return nil
}

func (p *ProcessCmd)  removeCredential () error {
	p.command.SysProcAttr.Token.Close()

	return nil
}

func impersonate(user string, pass string) error {
	token, err := logonUser(user, pass)
	if err != nil {
		return err
	}
	defer mustCloseHandle(token)

	if rc, _, ec := syscall.Syscall(impersonateProc.Addr(), 1, uintptr(token), 0, 0); rc == 0 {
		return error(ec)
	}
	return nil
}

func IsUserValid (userName string, password string) error {
	vm_password,err := util.GetSecretParam(password)
	if err != nil {
		return err
	}
	token, err := logonUser(userName, vm_password)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("Authentication failed for user %s with password", userName)
		return errors.New("UsernameOrPasswordInvalid")
	}
	defer mustCloseHandle(token)
	return nil
}

func logonUser(user, pass string) (token syscall.Handle, err error) {
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

	if rc, _, ec := syscall.Syscall6(logonProc.Addr(), 6,
		uintptr(unsafe.Pointer(&pu[0])),
		uintptr(unsafe.Pointer(&domain[0])),
		uintptr(unsafe.Pointer(&pp[0])),
		logon32LOGONINTERACTIVE,
		logon32ProviderDefault,
		uintptr(unsafe.Pointer(&token))); rc == 0 {
		err = error(ec)
	}
	return
}

func mustCloseHandle(handle syscall.Handle) {
	if err := syscall.CloseHandle(handle); err != nil {
		log.GetLogger().Errorln(err)
	}
}

//revertToSelf reverts the impersonation process.
func revertToSelf() error {
	if rc, _, ec := syscall.Syscall(revertSelfProc.Addr(), 0, 0, 0, 0); rc == 0 {
		return error(ec)
	}
	return nil
}