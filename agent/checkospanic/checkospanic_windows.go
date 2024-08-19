package checkospanic

import (
	"bytes"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
)

const (
	command            = "powershell"
	commandTimeout     = 10
	targetProviderName = "Microsoft-Windows-WER-SystemErrorReporting"
	// this script will find the latest event record provided by Microsoft-Windows-Kernel-General
	scriptMicrosoftWindowsWERSystemErrorReporting = ` $Events=Get-WinEvent -FilterHashtable @( @{ LogName='System' ; ProviderName='Microsoft-Windows-WER-SystemErrorReporting'; Id=1001; Level=2}) | Sort-Object TimeCreated  -Descending
 if ($Events.Length -gt 0){
    $item=$Events[0]
    echo "createTime:$($item.TimeCreated)"
    echo "message:$($item.Message)"
} 
`
)

var (
	bugcheckRegexp = regexp.MustCompile(`[^0-9a-f](0x[0-9a-f]{8})[^0-9a-f]`)
)

func ReportLastOsPanic() {
	logger := log.GetLogger().WithField("Phase", "ReportLastOsPanic")
	bugcheck, crashInfo, crashTime := FindWerSystemErrorReportingEvent(logger)
	if bugcheck == "" && crashInfo == "" {
		logger.Info("there is no event record need report")
		return
	}
	if time.Now().Sub(crashTime) > time.Hour*24 {
		logger.Info("the latest event record is 24 hours ago, ignore it")
		return
	}
	_, _, timeZone := timetool.NowWithTimezoneName()
	metrics.GetWindowsGuestOSPanicEvent(
		"bugcheck", bugcheck,
		"crashInfo", crashInfo,
		"crashTime", crashTime.Format("2006-01-02 15:04:05"),
		"crashTimeUTC", crashTime.UTC().Format("2006-01-02 15:04:05"),
		"timeZone", timeZone,
	).ReportEvent()
	logger.Info("the latest event record has reported")
}

// FindWerSystemErrorReportingEvent find latest event record provided by Microsoft-Windows-WER-SystemErrorReporting
// and parse fields `buckcheck` `crashInfo` `crashTime` from it
func FindWerSystemErrorReportingEvent(logger logrus.FieldLogger) (bugcheck, crashInfo string, crashTime time.Time) {
	processCmd := process.NewProcessCmd()
	var stdoutWrite bytes.Buffer
	var stderrWrite bytes.Buffer
	_, status, err := processCmd.SyncRun("", command, []string{"-command", scriptMicrosoftWindowsWERSystemErrorReporting}, &stdoutWrite, &stderrWrite, nil, nil, commandTimeout)
	if status == process.Timeout {
		logger.WithFields(logrus.Fields{
			"command": scriptMicrosoftWindowsWERSystemErrorReporting,
			"timeout": commandTimeout,
		}).Error("get windows event timeout")
		return
	} else if status == process.Fail || err != nil {
		logger.WithFields(logrus.Fields{
			"command": scriptMicrosoftWindowsWERSystemErrorReporting,
			"stdout":  stdoutWrite.String(),
			"stderr":  stderrWrite.String(),
			"err":     err,
		}).Error("get windows event failed")
		return
	}
	var content string
	if langutil.GetDefaultLang() != 0x409 {
		if tmp, err := langutil.GbkToUtf8(stdoutWrite.Bytes()); err != nil {
			logger.Error("GbkToUtf8 err: ", err)
		} else {
			content = string(tmp)
		}
	} else {
		content = stdoutWrite.String()
	}
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return
	}
	/*
		createTime:07/06/2023 19:25:08
		message:计算机已经从检测错误后重新启动。检测错误: 0x000000d1 (0xffff840001612010, 0x0000000000000002, 0x0000000000000000, 0xfffff801710a1981)。已将转储的数据保存在: C:\Windows\MEMORY.DMP。...
	*/
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "createTime:") {
			timeStr := line[len("createTime:"):]
			crashTime, err = time.ParseInLocation("1/2/2006 15:04:05", timeStr, time.Local)
			if err != nil {
				logger.WithFields(logrus.Fields{
					"line": line,
					"err":  err,
				}).Error("parse crash time failed")
				return
			}
		} else if strings.HasPrefix(line, "message:") {
			crashInfo = line[len("message:"):]
			if bugcheckRegexp.MatchString(crashInfo) {
				item := bugcheckRegexp.FindStringSubmatch(crashInfo)
				if len(item) != 2 {
					bugcheck = "not found"
				} else {
					bugcheck = item[1]
				}
			} else {
				bugcheck = "not found"
			}
		}
	}
	return
}
