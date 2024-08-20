//go:build !windows

package checkagentpanic

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

const (
	checkPoint = "Agent check point"

	stdoutFileName = "aliyun-service.out"
	stderrFileName = "aliyun-service.err"
)

// RedirectStdouterr redirect os.Stdout and os.Stderr to aliyun-service.err and
// aliyun-service.out in log directory if agent running in SysVinit or Upstart
// environment. In systemd environment stdout/stderr will be writen to journal 
// and console.
func RedirectStdouterr() {
	if util.IsSystemdLinux() {
		return
	}
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
	} else {
		os.Stderr = stderrF
		log.GetLogger().Infof("Redirect stderr to file: %s", stderrFile)
	}

	if stdoutF, err := os.OpenFile(stdoutFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm); err != nil {
		log.GetLogger().Errorf("Open file %s failed: %v", stdoutFile, err)
	} else {
		os.Stdout = stdoutF
		log.GetLogger().Infof("Redirect stdout to file: %s", stderrFile)
	}
}

func CheckAgentPanic() {
	defer fmt.Fprintln(os.Stderr, checkPoint)
	// lastPanicInfo will be empty if this is a systemd environment.
	if !lastPanicTimestamp.IsZero() && len(lastPanicInfo) > 0 {
		metrics.GetAgentLastPanicEvent(
			"panicTime", lastPanicTimestamp.Format("2006-01-02 15:04:05"),
			"panicInfo", lastPanicInfo,
		).ReportEvent()
	}
}
