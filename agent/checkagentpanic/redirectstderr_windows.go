//go:build windows

package checkagentpanic

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	stdoutFileName = "aliyun-service-out.txt"
	stderrFileName = "aliyun-service-err.txt"
)

var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	procSetStdHandle = kernel32.MustFindProc("SetStdHandle")
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

// RedirectStdouterr redirect os.Stdout and os.Stderr to aliyun-service-err.txt
// and aliyun-service-out.txt in log directory.
func RedirectStdouterr() {
	stdouterrDir, err := pathutil.GetLogPath()
	if err != nil {
		log.GetLogger().Errorf("Get log directory %s for stdout/stderr file failed: ", stdouterrDir)
		return
	}
	stdoutFile := filepath.Join(stdouterrDir, stdoutFileName)
	stderrFile := filepath.Join(stdouterrDir, stderrFileName)

	lastPanicInfo = searchPanicInfoFromFile(stderrFile)
	if len(lastPanicInfo) > 0 {
		finfo, err := os.Stat(stderrFile)
		if err != nil {
			log.GetLogger().Errorf("Get stderr file[%s] info failed: %v", stderrFile, err)
		}
		lastPanicTimestamp = finfo.ModTime()
	}

	if stderrF, err := os.OpenFile(stderrFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm); err != nil {
		log.GetLogger().Errorf("Open file %s failed: %v", stderrFile, err)
	} else if err = setStdHandle(syscall.STD_ERROR_HANDLE, syscall.Handle(stderrF.Fd())); err != nil {
		stderrF.Close()
		log.GetLogger().Errorf("Set STD_ERROR_HANDLE failed: %v", err)
	} else {
		os.Stderr = stderrF
		log.GetLogger().Infof("Redirect stderr to file: %s", stderrFile)
	}

	if stdoutF, err := os.OpenFile(stdoutFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm); err != nil {
		log.GetLogger().Errorf("Open file %s failed: %v", stdoutFile, err)
	} else if err = setStdHandle(syscall.STD_OUTPUT_HANDLE, syscall.Handle(stdoutF.Fd())); err != nil {
		stdoutF.Close()
		log.GetLogger().Errorf("Set STD_OUTPUT_HANDLE failed: %v", err)
	} else {
		os.Stdout = stdoutF
		log.GetLogger().Infof("Redirect stdout to file: %s", stdoutFile)
	}
}

func CheckAgentPanic() {
	lastPanicInfo = strings.TrimSpace(lastPanicInfo)
	if len(lastPanicInfo) == 0 {
		log.GetLogger().Info("Last agent panic Info is empty")
		return
	}
	metrics.GetAgentLastPanicEvent(
		"panicInfo", lastPanicInfo,
		"panicTime", lastPanicTimestamp.Format("2006-01-02 15:04:05"),
	).ReportEvent()
}
