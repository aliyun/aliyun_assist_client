//go:build linux

package perfmon

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/cgroup"
	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	cgroup_cfg        = "/usr/local/share/aliyun-assist/config/cgroup"
	cgroup_name       = "aliyun_assist_cpu"
	default_cpu_limit = 15
)

var (
	cpuCgroupLimit = false
)

func getCpuLimit() int64 {
	// /usr/local/share/aliyun-assist/config/cgroup
	var cpu_limit int64 = default_cpu_limit
	c, err := os.ReadFile(cgroup_cfg)
	if err == nil {
		i, err := strconv.ParseInt(strings.TrimSpace(string(c)), 10, 64)
		if err == nil {
			cpu_limit = i
			return cpu_limit
		}
	}
	os.MkdirAll(path.Dir(cgroup_cfg), os.ModePerm)
	os.WriteFile(cgroup_cfg, []byte(fmt.Sprintf("%d", cpu_limit)), 0644)
	return cpu_limit
}

func InitCgroup() error {
	c, e := cgroup.NewManager(os.Getpid(), cgroup_name, "cpu")
	if e != nil {
		return e
	}
	cpuLimit := getCpuLimit()
	log.GetLogger().Infoln("cpuLimit=", cpuLimit)
	cfg := &cgroup.Config{
		CpuQuota: int64(1000 * cpuLimit),
	}
	return c.Set(cfg)
}

func reachCpuOverloadLimit(cpuOverLoadCount int, cpuUsage float64) int {
	if !cpuCgroupLimit {
		if err := InitCgroup(); err == nil {
			log.GetLogger().Infoln("InitCgroup OK")
			report := clientreport.ClientReport{
				ReportType: "init_cgroup",
				Info:       fmt.Sprintf("cpu=%f", cpuUsage),
			}
			clientreport.SendReport(report)
			cpuCgroupLimit = true
			cpuOverLoadCount = 0
		} else {
			log.GetLogger().Infoln("InitCgroup error, so self kill...")
			report := clientreport.ClientReport{
				ReportType: "self_kill",
				Info:       fmt.Sprintf("cpu=%f", cpuUsage),
			}
			clientreport.SendReport(report)
			log.GetLogger().Fatalln("self kill for CPU Overload... CPU=", cpuUsage)
		}
	} else {
		// cgroup limit cpu not work, self kill
		report := clientreport.ClientReport{
			ReportType: "high_cpu",
			Info:       fmt.Sprintf("cpu=%f", cpuUsage),
		}
		clientreport.SendReport(report)
		log.GetLogger().Fatalln("self kill for CPU Overload... CPU=", cpuUsage)
	}
	return cpuOverLoadCount
}
