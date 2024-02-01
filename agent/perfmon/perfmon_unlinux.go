//go:build !linux

package perfmon

import (
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

func reachCpuOverloadLimit(cpuOverLoadCount int, cpuUsage float64) int {
	report := clientreport.ClientReport{
		ReportType: "self_kill",
		Info:       fmt.Sprintf("cpu=%f", cpuUsage),
	}
	clientreport.SendReport(report)
	log.GetLogger().Fatalln("self kill for CPU Overload... CPU=", cpuUsage)
	return cpuOverLoadCount
}
