package pluginmanager

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

func loadIntervalConf() {
	// 读取配置文件，如果有就读取并设置相关变量。用来测试的
	cpath, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(cpath))
	intervalPath := path.Join(dir, "config", "PluginCheckInterval")
	content, err := ioutil.ReadFile(intervalPath)
	if err != nil {
		return
	}
	contentStr := string(content)
	/*
		pluginHealthScanInterval=60,
		pluginHealthPullInterval=60,
		pluginUpdateCheckInterval=60
		pluginListReportInterval=60
	*/
	pvsList := strings.Split(contentStr, ",")
	for _, pvs := range pvsList {
		pvs = strings.TrimSpace(pvs)
		if len(pvs) == 0 {
			continue
		}
		pv := strings.Split(pvs, "=")
		if len(pv) != 2 {
			continue
		}
		interval, err := strconv.Atoi(pv[1])
		if err != nil {
			continue
		}
		setInterval(pv[0], interval)
	}
}

func setInterval(params string, interval int) {
	// if interval < 60 {
	// 	interval = 60
	// }
	// if interval > 2*60*60 {
	// 	interval = 2 * 60 * 60
	// }
	if params == "pluginHealthScanInterval" {
		pluginHealthScanInterval = interval
		log.GetLogger().Infof("pluginHealthScanInterval is init as %d seconds", interval)
	} else if params == "pluginHealthPullInterval" {
		pluginHealthPullInterval = interval
		log.GetLogger().Infof("pluginHealthPullInterval is init as %d seconds", interval)
	} else if params == "pluginUpdateCheckInterval" {
		pluginUpdateCheckInterval = interval
		log.GetLogger().Infof("pluginUpdateCheckIntervalSeconds is init as %d seconds", interval)
	} else if params == "pluginListReportInterval" {
		pluginListReportInterval = interval
		log.GetLogger().Infof("pluginListReportInterval is init as %d seconds", interval)
	}
}
