package checkospanic

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/common/executil"
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
	scriptLatestThreePatches = `(Get-HotFix -Description "Security Update"| sort-object -Descending {[int]($_.HotFixID -replace 'KB', '')}| Select-Object -First 3).HotFixID -join ','`
)

var (
	bugcheckRegexp = regexp.MustCompile(`[^0-9a-f](0x[0-9a-f]{8})[^0-9a-f]`)
)

func ReportLastOsPanic() {
	logger := log.GetLogger().WithField("Phase", "ReportLastOsPanic")
	bugcheck, crashInfo, latestPatches, crashTime := FindWerSystemErrorReportingEvent(logger)
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
		"latestPatches", latestPatches,
	).ReportEvent()
	logger.Info("the latest event record has reported")
}

// FindWerSystemErrorReportingEvent find latest event record provided by Microsoft-Windows-WER-SystemErrorReporting
// and parse fields `buckcheck` `crashInfo` `crashTime` from it
func FindWerSystemErrorReportingEvent(logger logrus.FieldLogger) (bugcheck, crashInfo, lastThreePatches string, crashTime time.Time) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*commandTimeout)
	defer cancel()

	cmd := executil.CommandWithContext(ctx, command, "-command", scriptMicrosoftWindowsWERSystemErrorReporting)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"command": scriptMicrosoftWindowsWERSystemErrorReporting,
		}).WithError(err).Error("get windows event failed")
		return
	}

	/*
		createTime:07/06/2023 19:25:08
		message:计算机已经从检测错误后重新启动。检测错误: 0x000000d1 (0xffff840001612010, 0x0000000000000002, 0x0000000000000000, 0xfffff801710a1981)。已将转储的数据保存在: C:\Windows\MEMORY.DMP。...
	*/
	output = transcoding(logger, output)
	lines := strings.Split(string(output), "\n")
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

	cmd = executil.CommandWithContext(ctx, command, "-command", scriptLatestThreePatches)
	output, err = cmd.CombinedOutput()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"command": scriptLatestThreePatches,
		}).WithError(err).Error("get latest patches failed")
		return
	}
	output = transcoding(logger, output)
	lastThreePatches = string(output)

	return
}

func transcoding(logger logrus.FieldLogger, data []byte) []byte {
	var res []byte
	var err error
	if langutil.GetDefaultLang() != 0x409 {
		if res, err = langutil.GbkToUtf8(data); err != nil {
			logger.Error("GbkToUtf8 err: ", err)
			res = data
		}
	} else {
		res = data
	}
	res = bytes.TrimSpace(res)
	return res
}
