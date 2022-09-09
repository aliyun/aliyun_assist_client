package checkkdump

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

const (
	// DefaultCheckIntervalSeconds is the default interval for report ecs-dump service status
	defaultCheckIntervalSeconds = 3600 * 24
)

func CheckKdumpTimer() error {
	timerManager := timermanager.GetTimerManager()
	timer, err := timerManager.CreateTimerInSeconds(doCheck, defaultCheckIntervalSeconds)
	if err != nil {
		return err
	}
	_, err = timer.Run()
	if err != nil {
		return err
	}
	return nil
}

func doCheck() {
	status, err := serviceStatus()
	if err != nil {
		log.GetLogger().Error("Get kdump service status err: ", err)
	}
	metrics.GetKdumpServiceStatusEvent(
		"status", status,
	).ReportEvent()
}

// 参考 http://gitlab.alibaba-inc.com/ecs-image/imgtools/blob/master/aliyun_plug/ecs_dump_config/windows/1.3/windump.ps1
func serviceStatus() (status string, err error) {
	status = "OFF"
	out := bytes.Buffer{}
	processCmd := process.NewProcessCmd()
	if _, _, err = processCmd.SyncRun("", "powershell", []string{"Test-Path \"HKLM:\\SOFTWARE\\Microsoft\\Windows NT\\CurrentVersion\\SoftwareProtectionPlatform\""}, &out, &out, nil, nil, 10); err != nil {
		return
	}
	rt := out.String()
	rt = strings.Replace(rt, "\n", "", -1)
	rt = strings.Replace(rt, "\r", "", -1)
	if rt == "True" {
		out = bytes.Buffer{}
		processCmd = process.NewProcessCmd()
		if _, _, err = processCmd.SyncRun("", "powershell", []string{"Get-ChildItem  $env:c:\\ -Force | Where-Object { $_.Name -eq \"pagefile.sys\" }"}, &out, &out, nil, nil, 10); err != nil {
			return
		}
		pageStatus := out.String()
		pageStatus = strings.Replace(pageStatus, "\n", "", -1)
		pageStatus = strings.Replace(pageStatus, "\r", "", -1)

		out = bytes.Buffer{}
		processCmd = process.NewProcessCmd()
		if _, _, err = processCmd.SyncRun("", "powershell", []string{"(Get-ItemProperty \"HKLM:\\SYSTEM\\CurrentControlSet\\Control\\CrashControl\" -Name CrashDumpEnabled).CrashDumpEnabled"}, &out, &out, nil, nil, 10); err != nil {
			return
		}
		levelStr := out.String()
		levelStr = strings.Replace(levelStr, "\n", "", -1)
		levelStr = strings.Replace(levelStr, "\r", "", -1)
		var level int
		level, err = strconv.Atoi(levelStr)
		if err != nil {
			return
		}
		if level > 0 && level <= 7 && pageStatus != "" {
			status = "ON"
		}
	}
	return
}
