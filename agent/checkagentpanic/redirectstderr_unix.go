//go:build !windows

package checkagentpanic

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func RedirectStderr() error { return nil }

const (
	checkPoint = "Agent check point"
)

func CheckAgentPanic() error {
	logger := log.GetLogger().WithField("function", "CheckAgentPanic")
	defer fmt.Fprintln(os.Stderr, checkPoint)
	var panicTime string
	var panicInfo []string
	if util.IsSystemdLinux() {
		panicTime, panicInfo = searchPanicInfoFromJournalctl(logger)
	} else {
		stderrLogPath, err := getStderrLogPath()
		if err != nil {
			logger.Error("get stderr log path failed: ", err)
			return err
		}
		if !util.CheckFileIsExist(stderrLogPath) {
			logger.Error("stderr log file not exist: ", stderrLogPath)
			return err
		}
		fileInfo, err := os.Stat(stderrLogPath)
		if err != nil {
			logger.Error("get file stat of stderr log file failed: ", err)
			return err
		}
		panicTime = fileInfo.ModTime().Format("2006-01-02 15:04:05")
		stderrFile, err := os.Open(stderrLogPath)
		if err != nil {
			logger.Error("open stderr log file failed: ", err)
			return err
		}
		var ispanicInfo bool
		reader := bufio.NewReader(stderrFile)
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			line = strings.TrimRight(line, "\n")
			// find the latest panic info
			if strings.HasPrefix(line, checkPoint) {
				ispanicInfo = false
				panicInfo = panicInfo[:0]
				continue
			}
			if ispanicInfo {
				panicInfo = append(panicInfo, line)
			} else if strings.HasPrefix(line, "panic:") {
				ispanicInfo = true
				panicInfo = append(panicInfo, line)
			}
		}
	}
	if len(panicInfo) == 0 {
		logger.Info("not found panic information")
		return nil
	}
	metrics.GetAgentLastPanicEvent(
		"panicTime", panicTime,
		"panicInfo", strings.Join(panicInfo, "\n"),
	).ReportEvent()
	return nil
}
