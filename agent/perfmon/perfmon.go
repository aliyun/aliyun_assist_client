package perfmon

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"
	"math/rand"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/update"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/shirou/gopsutil/v3/process"
)

const (
	CPU_LIMIT      = 20.0
	MEM_LIMIT      = 1024 * 1024 * 50 // byte
	OVERLAOD_LIMIT = 3
)

var (
	perfMonitorIntervalSecond = 5

	cpuPercentTotal int64 // store cpu usage percent by int64, 12% is stored as 1200
	cpuPercentMax   int64
	cpuCount        int64
	memRssTotal     int64
	memRssMax       int64
	memCount        int64
	dataLock        sync.RWMutex

	cpu_overload_count int = 0
	mem_overload_count int = 0
)

// Periodically collect its own CPU and memory load
func generatePerfData(logger logrus.FieldLogger) error {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		logger.Error("get process failed: ", err)
		return err
	}
	_, err = p.Percent(0)
	if err != nil {
		logger.Error("get process cpu percent failed: ", err)
		return err
	}

	tick := time.NewTicker(time.Duration(perfMonitorIntervalSecond) * time.Second)
	defer tick.Stop()
	for {
		<-tick.C
		dataLock.Lock()
		cpuPercent, err := p.Percent(0)
		if err != nil {
			logger.Error("get cpu percent failed: ", err)
			cpuPercent = -1
		} else {
			tmpCpuPercent := int64(cpuPercent * 100)
			cpuPercentTotal += tmpCpuPercent
			if tmpCpuPercent > cpuPercentMax {
				cpuPercentMax = tmpCpuPercent
			}
			cpuCount += 1
		}
		var memRss int64
		memInfo, err := p.MemoryInfo()
		if err != nil {
			logger.Error("get cpu percent failed: ", err)
			memRss = -1
		} else {
			memRssTotal += int64(memInfo.RSS)
			if memInfo.RSS > uint64(memRssMax) {
				memRssMax = int64(memInfo.RSS)
			}
			memRss = int64(memInfo.RSS)
			memCount += 1
		}
		dataLock.Unlock()
		go checkCpuMemLoad(cpuPercent, memRss)
	}
}

func StartSelfKillMon() {
	logger := log.GetLogger().WithField("Phase", "perfMonitor")
	go generatePerfData(logger)

	timerManager := timermanager.GetTimerManager()
	timer, err := timerManager.CreateTimerInSeconds(func() {
		dataLock.Lock()
		defer func() {
			cpuPercentTotal = 0
			cpuCount = 0
			memRssTotal = 0
			memCount = 0
			cpuPercentMax = 0
			memRssMax = 0
			dataLock.Unlock()
		}()
		var cpuAvg, memAvg int64
		if cpuCount >= 1 {
			cpuAvg = cpuPercentTotal / cpuCount
		} else {
			cpuAvg = -1
		}
		if memCount >= 1 {
			memAvg = memRssTotal / memCount
		} else {
			memAvg = -1
		}
		metrics.GetPerfSampleEvent(
			"cpuAvg", strconv.FormatInt(cpuAvg, 10),
			"cpuMax", strconv.FormatInt(cpuPercentMax, 10),
			"memAvg", strconv.FormatInt(memAvg, 10),
			"memMax", strconv.FormatInt(memRssMax, 10),
		).ReportEvent()
	}, 3600*24)
	if err != nil {
		logger.Error("create timer for perform report failed: ", err)
		return
	}
	mutableSchedule, ok := timer.Schedule.(*timermanager.MutableScheduled)
	if !ok {
		logger.Error("unexpected schedule type of perform report timer")
		return
	}
	mutableSchedule.NotImmediately()
	_, err = timer.Run()
	if err != nil {
		logger.Error("start timer for perform report failed: ", err)
		return
	}

}

// Check whether its own cpu and memory load exceeds the limit
func checkCpuMemLoad(cpuUsage float64, memory int64) {
	var _taskFactory *taskengine.TaskFactory = taskengine.GetTaskFactory()
	if _taskFactory.IsAnyTaskRunning() || update.IsCPUIntensiveActionRunning() || taskengine.GetSessionFactory().IsAnyTaskRunning() { //没有任务执行时才监控性能
		return
	}
	if statemanager.IsStateManagerTimerRunning() || statemanager.IsStateConfigTimerRunning() {
		// 拉取并解析终态配置时、应用或监控终态配置时不监控性能
		return
	}
	if cpuUsage >= CPU_LIMIT {
		cpu_overload_count += 1
		go func(cpuUsageNow float64, cpuOverLoadCount int) {
			var profileBuf bytes.Buffer
			var cpuProfile, cpuProfileErr string
			// pprof.StartCPUProfile will return err if profiling is already enabled.
			if rand.Intn(10000) > 100 {
				cpuProfileErr = "only sampe cpu profile with a probability of 1/100"
			} else if err := pprof.StartCPUProfile(&profileBuf); err == nil {
				time.Sleep(10 * time.Second)
				pprof.StopCPUProfile()
				cpuProfile = base64.StdEncoding.EncodeToString(profileBuf.Bytes())
			} else {
				cpuProfileErr = err.Error()
			}
			metrics.GetCpuOverloadEvent(
				"cpu", fmt.Sprintf("%.2f", cpuUsageNow),
				"info", fmt.Sprintf("CPU Overload... CPU=%.2f", cpuUsageNow),
				"count", strconv.Itoa(cpuOverLoadCount),
				"cpuProfile", cpuProfile,
				"cpuProfileErr", cpuProfileErr,
			).ReportEvent()
		}(cpuUsage, cpu_overload_count)
		log.GetLogger().Infoln("CPU Overload... CPU=", cpuUsage)
	} else {
		cpu_overload_count = 0
	}
	if memory >= MEM_LIMIT {
		// 上报memStats
		mem_overload_count += 1
		memStats := &runtime.MemStats{}
		runtime.ReadMemStats(memStats)
		metrics.GetMemOverloadEvent(
			"mem", fmt.Sprintf("%d", memory),
			"info", fmt.Sprintf("Memory Overload... MEM=%d", memory),
			"count", strconv.Itoa(mem_overload_count),
			"HeapAlloc", strconv.FormatUint(memStats.HeapAlloc, 10),
			"HeapIdle", strconv.FormatUint(memStats.HeapIdle, 10),
			"HeapInuse", strconv.FormatUint(memStats.HeapInuse, 10),
			"HeapReleased", strconv.FormatUint(memStats.HeapReleased, 10),
			"StackInuse", strconv.FormatUint(memStats.StackInuse, 10),
		).ReportEvent()
		log.GetLogger().Infoln("Memory Overload... MEM=", memory)
	} else {
		mem_overload_count = 0
	}
	if cpu_overload_count >= OVERLAOD_LIMIT {
		cpu_overload_count = reachCpuOverloadLimit(cpu_overload_count, cpuUsage)
	}
	if mem_overload_count >= OVERLAOD_LIMIT {
		report := clientreport.ClientReport{
			ReportType: "self_kill",
			Info:       fmt.Sprintf("mem=%f", float64(memory)),
		}
		clientreport.SendReport(report)
		log.GetLogger().Fatalln("self kill for Memory Overload... Mem=", memory)
	}
}
