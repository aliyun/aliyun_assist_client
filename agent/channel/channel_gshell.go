package channel

import (
	"encoding/json"
	"errors"
	"fmt"

	"os"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/apiserver"
)

const (
	defaultGshellMaxFrequencyCount  = 10
	defaultGshellMaxFrequencyPeriod = 10
)

var (
	// limit gshell kick_vm to occur up to gshellMaxFrequencyCount times in gshellMaxFrequencyPeriod second
	gshellMaxFrequencyCount  = defaultGshellMaxFrequencyCount  // 0 means no limit
	gshellMaxFrequencyPeriod = defaultGshellMaxFrequencyPeriod // the unit is seconds
	gshellMaxFrequencyMutex  sync.Mutex

	intervalToOpenNoGshellChannel = time.Minute
)

type GshellChannel struct {
	*Channel
	hGshell       *os.File
	WaitCheckDone sync.WaitGroup

	kickvmTimes     []*time.Time
	kickvmTimeStart int
	kickvmTimeEnd   int
}

type gshellStatus struct {
	Code             int64  `json:"code"`
	GshellSupport    string `json:"gshellSupport"`
	InstanceID       string `json:"instanceId"`
	RequestID        string `json:"requestId"`
	Retry            int64  `json:"retry"`
	ThrottlingConfig struct {
		MaxKickVmCount   int `json:"maxKickVmCount"`
		MaxKickVmPeriod  int `json:"maxKickVmPeriod"`
		WssCoolDownCount int `json:"wssCoolDownCount"`
		WssCoolDownTime  int `json:"wssCoolDownTime"`
	} `json:"throttlingConfig"`
}

func (c *GshellChannel) IsSupported() bool {
	if apiserver.IsHybrid() {
		return false
	}
	if !c.Working.IsSet() {
		if c.startChannelUnsafe() != nil {
			return false
		}
	}
	url := util.GetGshellCheckService()
	resp, err := util.HttpPost(url, "", "text")
	if err != nil {
		log.GetLogger().Errorln("HttpPost ", url, "error! ", err)
		return false
	}
	log.GetLogger().Infoln("HttpPost ", url, "OK! resp: ", resp)

	var gstatus gshellStatus
	if err := json.Unmarshal([]byte(resp), &gstatus); err != nil {
		return false
	}
	c.updateMaxFrequency(gstatus.ThrottlingConfig.MaxKickVmCount, gstatus.ThrottlingConfig.MaxKickVmPeriod)
	wssCoolDownCount = gstatus.ThrottlingConfig.WssCoolDownCount
	wssCoolDownTime = gstatus.ThrottlingConfig.WssCoolDownTime

	return gstatus.GshellSupport == "true"
}

// func (c *GshellChannel) canOpenGshell() bool {
// 	gShellSupport := false
// 	c.lock.Lock()
// 	defer c.lock.Unlock()
// 	gshellPath := "/dev/virtio-ports/org.qemu.guest_agent.0"
// 	if runtime.GOOS == "windows" {
// 		gshellPath = "\\\\.\\Global\\org.qemu.guest_agent.0"
// 	}
// 	if f, e := os.Open(gshellPath); e == nil {
// 		gShellSupport = true
// 		f.Close()
// 	}
// 	log.GetLogger().Infoln("open gshell status:", gshellPath, gShellSupport)
// 	return gShellSupport
// }

func (c *GshellChannel) StartChannel() error {
	if c.Working.IsSet() {
		return nil
	}
	return c.startChannelUnsafe()
}

func (c *GshellChannel) startChannelUnsafe() error {
	if !c.Working.CompareAndSwap(false, true) {
		log.GetLogger().Warning("startChannelUnsafe run duplicated, return it")
		return nil // startChannelUnsage 同一时间只能有一个执行
	}
	var gshellPath string
	var e error
	var h *os.File
	gshellPath, e = getGshellPath()
	if e == nil {
		h, e = os.OpenFile(gshellPath, os.O_RDWR, 0666)
	}
	if e != nil {
		metrics.GetChannelFailEvent(
			metrics.EVENT_SUBCATEGORY_CHANNEL_GSHELL,
			"errormsg", fmt.Sprintf("open gshell failed: %s  error: %s", gshellPath, e),
			"filepath", gshellPath,
			"type", ChannelTypeStr(c.ChannelType),
		).ReportEvent()
		log.GetLogger().Errorln("open gshell failed:", gshellPath, "error:", e)
		c.Working.Clear()
		return e
	}
	log.GetLogger().Infoln("open gshell ok:", gshellPath)
	c.hGshell = h
	c.WaitCheckDone.Add(1)
	go func() {
		defer c.Working.Clear()
		defer c.hGshell.Close()
		defer c.WaitCheckDone.Done()
		tick := time.NewTicker(time.Duration(200) * time.Millisecond)
		defer tick.Stop()
		buf := make([]byte, 2048)
		var lastKickvmFreqExceedTime time.Time
		for {
			<-tick.C
			n, err := c.hGshell.Read(buf)
			if err == nil && n > 0 {
				reachedFrequencyLimit, count, period := c.hasReachFrequencyLimit()
				retStr := c.CallBack(string(buf[:n]), ChannelGshellType)
				if len(retStr) > 0 {
					log.GetLogger().Infoln("write:", retStr)
					_, err = c.hGshell.Write([]byte(retStr + "\n"))
					if err != nil {
						log.GetLogger().Errorln("write error:", err)
						report := clientreport.ClientReport{
							ReportType: "switch_channel_in_gshell",
							Info:       fmt.Sprintf("start switch :" + err.Error()),
						}
						clientreport.SendReport(report)
						go c.SwitchChannel()
						return
					}
				}
				if reachedFrequencyLimit && (time.Since(lastKickvmFreqExceedTime)>intervalToOpenNoGshellChannel) {
					lastKickvmFreqExceedTime = time.Now()
					tip := fmt.Sprintf("gshell kick_vm has reached max frequency %d times during %d seconds, " + 
						"try to open no-gshell channel", count, period)
					log.GetLogger().Info(tip)
					report := clientreport.ClientReport{
						ReportType: "switch_channel_in_gshell",
						Info:       fmt.Sprintf("start switch :" + tip),
					}
					clientreport.SendReport(report)
					// just try to open no-gshell cahnnel, do not close gshell self
					go c.openOtherChannel()
				}
			}
		}
	}()
	return nil
}

func (c *GshellChannel) StopChannel() error {
	c.WaitCheckDone.Wait()
	return nil
}

func (c *GshellChannel) SwitchChannel() error {
	c.StopChannel()
	time.Sleep(time.Duration(1) * time.Second)
	err := G_ChannelMgr.SelectAvailableChannel(ChannelGshellType)
	if err != nil {
		for i := 0; i < 5; i++ {
			if G_ChannelMgr.SelectAvailableChannel(ChannelNone) == nil {
				metrics.GetChannelSwitchEvent(
					"type", ChannelTypeStr(G_ChannelMgr.GetCurrentChannelType()),
					"info", fmt.Sprintf("success: Current channel is %d", G_ChannelMgr.GetCurrentChannelType()),
					"reportType", "switch_channel_in_gshell",
				).ReportEvent()

				report := clientreport.ClientReport{
					ReportType: "switch_channel_in_gshell",
					Info:       fmt.Sprintf("success: Current channel is %d", G_ChannelMgr.GetCurrentChannelType()),
				}
				clientreport.SendReport(report)
				return nil
			}
			time.Sleep(time.Duration(5) * time.Second)
		}
	} else {
		metrics.GetChannelSwitchEvent(
			"type", ChannelTypeStr(G_ChannelMgr.GetCurrentChannelType()),
			"info", fmt.Sprintf("success: Current channel is %d", G_ChannelMgr.GetCurrentChannelType()),
			"reportType", "switch_channel_in_gshell",
		).ReportEvent()

		report := clientreport.ClientReport{
			ReportType: "switch_channel_in_gshell",
			Info:       fmt.Sprintf("success: Current channel is %d", G_ChannelMgr.GetCurrentChannelType()),
		}
		clientreport.SendReport(report)
		return nil
	}
	metrics.GetChannelSwitchEvent(
		"type", "type", ChannelTypeStr(G_ChannelMgr.GetCurrentChannelType()),
		"info", fmt.Sprintf("fail: no available channel"),
		"reportType", "switch_channel_in_gshell",
	).ReportEvent()

	report := clientreport.ClientReport{
		ReportType: "switch_channel_in_gshell",
		Info:       fmt.Sprintf("fail: no available channel"),
	}
	clientreport.SendReport(report)
	return errors.New("no available channel")
}

func (c *GshellChannel) openOtherChannel() error {
	if err := G_ChannelMgr.SelectAvailableChannel(ChannelGshellType); err != nil {
		log.GetLogger().Error("open other no-gshell channel failed: ", err)
	} else {
		log.GetLogger().Infof("open other no-gshell channel[%s] success", ChannelTypeStr(G_ChannelMgr.ActiveChannel.GetChannelType()))
	}
	return errors.New("no available channel")
}

// hasReachFrequencyLimit: check if gshell kick_vm reaches frequency limit
func (c *GshellChannel) hasReachFrequencyLimit() (bool, int, int) {
	if gshellMaxFrequencyCount == 0 {
		return false, 0, 0
	}
	gshellMaxFrequencyMutex.Lock()
	defer gshellMaxFrequencyMutex.Unlock()
	now := time.Now()
	if c.kickvmTimeStart == -1 {
		c.kickvmTimes[0] = &now
		c.kickvmTimeStart = 0
		c.kickvmTimeEnd = 0
		return false, gshellMaxFrequencyCount, gshellMaxFrequencyPeriod
	}
	c.kickvmTimeStart = (c.kickvmTimeStart + 1) % gshellMaxFrequencyCount
	if c.kickvmTimeStart == c.kickvmTimeEnd {
		lastTimepoint := c.kickvmTimes[c.kickvmTimeEnd]
		c.kickvmTimes[c.kickvmTimeStart] = &now
		c.kickvmTimeEnd = (c.kickvmTimeEnd + 1) % gshellMaxFrequencyCount
		if lastTimepoint.Add(time.Second * time.Duration(gshellMaxFrequencyPeriod)).After(now) {
			return true, gshellMaxFrequencyCount, gshellMaxFrequencyPeriod
		}
	}
	c.kickvmTimes[c.kickvmTimeStart] = &now
	return false, gshellMaxFrequencyCount, gshellMaxFrequencyPeriod
}

func (c *GshellChannel) updateMaxFrequency(count, period int) {
	gshellMaxFrequencyMutex.Lock()
	defer gshellMaxFrequencyMutex.Unlock()
	if gshellMaxFrequencyCount != count {
		if count <= 0 {
			gshellMaxFrequencyCount = 0
			c.kickvmTimes = nil
		} else {
			gshellMaxFrequencyCount = count
			c.kickvmTimes = make([]*time.Time, gshellMaxFrequencyCount)
		}
		c.kickvmTimeStart = -1
		c.kickvmTimeEnd = -1
	}
	gshellMaxFrequencyPeriod = period
}

func NewGshellChannel(CallBack OnReceiveMsg) IChannel {
	g := &GshellChannel{
		Channel: &Channel{
			CallBack:    CallBack,
			ChannelType: ChannelGshellType,
		},
		kickvmTimes:     make([]*time.Time, gshellMaxFrequencyCount),
		kickvmTimeStart: -1,
		kickvmTimeEnd:   -1,
	}
	g.Working.Clear()
	return g
}
