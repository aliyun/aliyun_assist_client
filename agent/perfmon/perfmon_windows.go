package perfmon

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

type FILETIME struct {
	DwLowDateTime  uint32
	DwHighDateTime uint32
}

var (
	Modkernel32        = windows.NewLazySystemDLL("kernel32.dll")
	ProcGetSystemTimes = Modkernel32.NewProc("GetSystemTimes")
)

func (p *procStat) UpdateSysStat() error {
	var lpIdleTime FILETIME
	var lpKernelTime FILETIME
	var lpUserTime FILETIME
	r, _, _ := ProcGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&lpIdleTime)),
		uintptr(unsafe.Pointer(&lpKernelTime)),
		uintptr(unsafe.Pointer(&lpUserTime)))
	if r == 0 {
		return windows.GetLastError()
	}
	p.systotal = 4294967296*uint64(lpKernelTime.DwHighDateTime) + uint64(lpKernelTime.DwLowDateTime)
	p.systotal = p.systotal + 4294967296*uint64(lpUserTime.DwHighDateTime) + uint64(lpUserTime.DwLowDateTime)
	return nil
}

func (p *procStat) UpdatePidStatInfo() error {
	var c windows.Handle
	var err error = nil
	if os.Getpid() != p.pid {
		c, err = windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(p.pid))
	} else {
		c, err = windows.GetCurrentProcess()
	}
	if err != nil {
		return err
	}
	if os.Getpid() != p.pid {
		defer windows.CloseHandle(c)
	}
	var CreationTime windows.Filetime
	var ExitTime windows.Filetime
	var KernelTime windows.Filetime
	var UserTime windows.Filetime

	if err = windows.GetProcessTimes(c, &CreationTime, &ExitTime, &KernelTime, &UserTime); err != nil {
		return err
	}

	p.utime = 4294967296*uint64(KernelTime.HighDateTime) + uint64(KernelTime.LowDateTime)
	p.stime = 4294967296*uint64(UserTime.HighDateTime) + uint64(UserTime.LowDateTime)
	//p.threads = readUInt(p.splitParts[19])
	//p.rss = readUInt(p.splitParts[23])
	return nil
}

func InitCgroup() error {
	return nil
}

func GetAgentCpuLoadWithTop(times int) (error, float64) {
	return errors.New("not supported"), 0.0
}
