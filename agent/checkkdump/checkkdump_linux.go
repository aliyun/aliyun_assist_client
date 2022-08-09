package checkkdump

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

const (
	// defaultCheckIntervalSeconds is the default interval for report ecs-dump service status
	defaultCheckIntervalSeconds = 3600 * 24
)

const (
	LINUX_SYSTEMD int = iota
	LINUX_SYSV
)
// 参考 http://gitlab.alibaba-inc.com/ecs-image/imgtools/blob/master/aliyun_plug/ecs_dump_config/linux/1.5/kdump.sh
var distroMap map[string]string = map[string]string{
	"Debian":        "debian",
	"Ubuntu":        "ubuntu",
	"SUSE Linux":    "suse",
}
var serviceType int         // linux服务管理系统的类型 systemd/sysv
var distribution string     // linux发行版
var majorVer string         // linux系统主版本号
var kdumpServiceName string // kdump服务名称

func CheckKdumpTimer() error {
	if util.IsSystemdLinux() {
		serviceType = LINUX_SYSTEMD
	} else {
		serviceType = LINUX_SYSV
	}
	// 确定 linux 发行版
	platFormName, err := osutil.OriginPlatformName()
	if err != nil {
		log.GetLogger().Warn("Get platFormName err: ", err)
		err = nil
	}
	for key, value := range distroMap {
		if strings.Contains(platFormName, key) {
			distribution = value
			break
		}
	}
	// 确定kdump的服务名称
	kdumpServiceName = "kdump"
	majorVer, _ = getLinuxOsMajorVer()
	if distribution == "ubuntu" || distribution == "debian" {
		kdumpServiceName = "kdump-tools"
	} else if distribution == "suse" {
		if majorVer == "11" {
			kdumpServiceName = "boot.kdump"
		}
	}
	// 创建定时器
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

// 参考 http://gitlab.alibaba-inc.com/ecs-image/imgtools/blob/master/aliyun_plug/ecs_dump_config/linux/1.5/kdump.sh
func serviceStatus() (status string, err error) {
	status = "OFF"
	var out bytes.Buffer
	var cmd string
	// check kdump installed
	// debian, ubuntu
	if serviceType == LINUX_SYSTEMD {
		// systemd
		out = bytes.Buffer{}
		cmd = fmt.Sprintf("systemctl  is-enabled  %s", kdumpServiceName)
		processCmd := process.NewProcessCmd()
		if _, _, err = processCmd.SyncRun("", "bash", []string{"-c", cmd}, &out, &out, nil, nil, 10); err != nil {
			return
		}
		rt := out.String()
		rt = strings.Replace(rt, "\n", "", -1)
		if rt == "enabled" {
			status = "ON"
		}
	} else {
		// sysv
		out = bytes.Buffer{}
		cmd = fmt.Sprintf("chkconfig --list  %s |grep -q on", kdumpServiceName)
		processCmd := process.NewProcessCmd()
		var exitCode int
		if exitCode, _, err = processCmd.SyncRun("", "bash", []string{"-c", cmd}, &out, &out, nil, nil, 10); err != nil {
			return
		}
		if exitCode == 0 {
			status = "ON"
		}
	}
	return
}

func getLinuxOsMajorVer() (majorVer string, err error) {
	var out bytes.Buffer
	processCmd := process.NewProcessCmd()
	if _, _, err = processCmd.SyncRun("", "bash", []string{"-c", "lsb_release -r 2>/dev/null |awk '{print $NF}' |awk -F. '{print $1}'"}, &out, &out, nil, nil, 10); err != nil {
		return
	}
	majorVer = out.String()
	majorVer = strings.Replace(majorVer, "\n", "", -1)
	return
}
