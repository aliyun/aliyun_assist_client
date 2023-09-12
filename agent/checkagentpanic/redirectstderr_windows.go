//go:build windows

package checkagentpanic

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
)

var (
	lastPanicInfo      string
	lastPanicTimestamp time.Time
)

func setStdHandle(stdhandle int32, handle syscall.Handle) error {
	r0, _, e1 := syscall.Syscall(procSetStdHandle.Addr(), 2, uintptr(stdhandle), uintptr(handle), 0)
	if r0 == 0 {
		if e1 != 0 {
			return error(e1)
		}
		return syscall.EINVAL
	}
	return nil
}

// RedirectStderr must be call after log.InitLog
func RedirectStderr() error {
	// read last panic output
	panicFile := filepath.Join(log.Logdir, "log", "panic.txt")
	content, err := os.ReadFile(panicFile)
	if err != nil {
		log.GetLogger().WithField("path", panicFile).Error("read last panic file failed: ", err)
	} else {
		lastPanicInfo = string(content)
		finfo, err := os.Stat(panicFile)
		if err != nil {
			log.GetLogger().WithField("path", panicFile).Error("get fileInfo of last panic file failed: ", err)
		} else {
			winFileAttr := finfo.Sys().(*syscall.Win32FileAttributeData)
			lastPanicTimestamp = time.Unix(0, winFileAttr.LastWriteTime.Nanoseconds())
			log.GetLogger().WithFields(logrus.Fields{
				"path": panicFile,
				"time": lastPanicTimestamp,
			}).Info("get modified time of last panic file")
		}
	}
	f, err := os.OpenFile(panicFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return err
	}
	// defer close(f)
	err = setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(f.Fd()))
	if err != nil {
		return err
	}
	os.Stderr = f
	return nil
}

func CheckAgentPanic() error {
	lastPanicInfo = strings.TrimSpace(lastPanicInfo)
	if len(lastPanicInfo) == 0 {
		log.GetLogger().Info("Last agent panic Info is empty")
		return nil
	}
	metrics.GetAgentLastPanicEvent(
		"panicInfo", lastPanicInfo,
		"panicTime", lastPanicTimestamp.Format("2006-01-02 15:04:05"),
	).ReportEvent()
	return nil
}
