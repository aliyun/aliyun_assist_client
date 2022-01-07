package perfmon

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
	"github.com/aliyun/aliyun_assist_client/agent/update"
)

type procStat struct {
	pid             int
	splitParts      []string
	utime           uint64
	stime           uint64
	rss             uint64
	threads         uint64
	systotal        uint64
	interval        uint
	callback        PerfCallback
	stopChanelEvent chan struct{}
	err             error
	cpu             int
}

type PerfCallback func(cpusage float64, memoryInKbs uint64, threads uint64)

func (p *procStat) generatePerfData() {
	err := p.UpdatePidStatInfo()
	if err != nil {
		p.err = err
		return
	}
	err = p.UpdateSysStat()
	if err != nil {
		p.err = err
		return
	}
	prevProcTime := p.utime + p.stime
	PrevSysTime := p.systotal
	tick := time.NewTicker(time.Duration(p.interval) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-p.stopChanelEvent:
			break
		case <-tick.C:
			err = p.UpdatePidStatInfo()
			if err != nil {
				p.err = err
				return
			}
			err = p.UpdateSysStat()
			if err != nil {
				p.err = err
				return
			}
			CurProcTime := p.utime + p.stime
			CurSysTime := p.systotal
			cpuasge := (float64(CurProcTime-prevProcTime) / float64(CurSysTime-PrevSysTime)) * 100 * float64(p.cpu)
			mem := p.rss
			thread := p.threads
			p.callback(cpuasge, mem, thread)
			prevProcTime = CurProcTime
			PrevSysTime = CurSysTime
		}
	}
}

func StartPerfmon(pid int, interval uint, callback PerfCallback) *procStat {
	p := &procStat{
		pid:             pid,
		callback:        callback,
		splitParts:      make([]string, 52),
		interval:        interval,
		stopChanelEvent: make(chan struct{}),
		cpu:             1,
	}
	if runtime.GOOS != "windows" {
		p.cpu = runtime.NumCPU()
	}
	go p.generatePerfData()
	return p
}

func (p *procStat) StopPerfmon() {
	p.stopChanelEvent <- struct{}{}
}

var _perf *procStat

const CPU_LIMIT = 20.0
const MEM_LIMIT = 1024 * 50
const OVERLAOD_LIMIT = 3

var cpu_overload_count int = 0
var mem_overload_count int = 0
var bCollectCpuLoadByAgent bool = true

func StartSelfKillMon() {
	var _taskFactory *taskengine.TaskFactory = taskengine.GetTaskFactory()
	_perf = StartPerfmon(os.Getpid(), 5, func(cpuUsage float64, memory uint64, threads uint64) {
		if _taskFactory.IsAnyTaskRunning() || update.IsCPUIntensiveActionRunning() || taskengine.GetSessionFactory().IsAnyTaskRunning() { //没有任务执行时才监控性能
			return
		}
		if statemanager.IsStateManagerTimerRunning() || statemanager.IsStateConfigTimerRunning() {
			// 拉取并解析终态配置时、应用或监控终态配置时不监控性能
			return
		}
		if cpuUsage >= CPU_LIMIT && bCollectCpuLoadByAgent {
			metrics.GetCpuOverloadEvent(
				"cpu", fmt.Sprintf("%.2f", cpuUsage),
				"info", fmt.Sprintf("CPU Overload... CPU=%.2f", cpuUsage),
			).ReportEvent()
			log.GetLogger().Infoln("CPU Overload... CPU=", cpuUsage)
			cpu_overload_count += 1
		} else {
			cpu_overload_count = 0
		}
		if memory >= MEM_LIMIT {
			metrics.GetMemOverloadEvent(
				"mem", fmt.Sprintf("%d", memory),
				"info", fmt.Sprintf("Memory Overload... MEM=%d", memory),
			).ReportEvent()
			log.GetLogger().Infoln("Memory Overload... MEM=", memory)
			mem_overload_count += 1
		} else {
			mem_overload_count = 0
		}
		if cpu_overload_count >= OVERLAOD_LIMIT && bCollectCpuLoadByAgent {
			if runtime.GOOS == "linux" {
				//将cpu_overload_coun将置0，因为top命令在采集了
				cpu_overload_count = 0
				bCollectCpuLoadByAgent = false
				go func() {
					//top命令采集当前CPU
					err, cpuByTop := GetAgentCpuLoadWithTop(1)
					if err == nil && cpuByTop >= CPU_LIMIT {
						//top命令采集的cpu信息和agent一致
						metrics.GetCpuOverloadEvent(
							"cpu", fmt.Sprintf("%.2f", cpuByTop),
							"info", fmt.Sprintf("CPU Overload by top... CPU=%.2f", cpuByTop),
						).ReportEvent()

						log.GetLogger().Infoln("CPU Overload by top... CPU=", cpuByTop)
						err = InitCgroup()
						if err == nil {
							bCollectCpuLoadByAgent = true
							log.GetLogger().Infoln("InitCgroup OK")
							report := clientreport.ClientReport{
								ReportType: "init_cgroup",
								Info:       fmt.Sprintf("cpu=%f", cpuUsage),
							}
							clientreport.SendReport(report)
							return
						}
						log.GetLogger().Infoln("InitCgroup error, so self kill...")
						report := clientreport.ClientReport{
							ReportType: "self_kill",
							Info:       fmt.Sprintf("cpu=%f", cpuUsage),
						}
						clientreport.SendReport(report)
						log.GetLogger().Fatalln("self kill for CPU Overload... CPU=", cpuUsage)
					}
					bCollectCpuLoadByAgent = true
				}()
				report := clientreport.ClientReport{
					ReportType: "high_cpu",
					Info:       fmt.Sprintf("cpu=%f", cpuUsage),
				}
				clientreport.SendReport(report)
			} else {
				report := clientreport.ClientReport{
					ReportType: "self_kill",
					Info:       fmt.Sprintf("cpu=%f", cpuUsage),
				}
				clientreport.SendReport(report)
				log.GetLogger().Fatalln("self kill for CPU Overload... CPU=", cpuUsage)
			}

		}
		if mem_overload_count >= OVERLAOD_LIMIT {
			report := clientreport.ClientReport{
				ReportType: "self_kill",
				Info:       fmt.Sprintf("mem=%f", float64(memory)),
			}
			clientreport.SendReport(report)
			log.GetLogger().Fatalln("self kill for Memory Overload... Mem=", memory)
		}
	})
}

func StopSelfKillMon() {
	if _perf != nil {
		_perf.StopPerfmon()
		_perf = nil
	}
}
