package metrics

import (
	"encoding/json"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
)

type MetricsEventID string
type EventLevel string
type EventCategory string
type EventSubCategory string

const (
	// event id
	EVENT_CHANNEL_FAILED    MetricsEventID = "agent.channel.failed"
	EVENT_CHANNEL_SWITCH    MetricsEventID = "agent.channel.switch"
	EVENT_UPDATE_FAILED     MetricsEventID = "agent.update.failed"
	EVENT_TASK_FAILED       MetricsEventID = "agent.task.failed"
	EVENT_TASK_WARN			MetricsEventID = "agent.task.warn"
	EVENT_HYBRID_REGISTER   MetricsEventID = "agent.hybrid.register"
	EVENT_HYBRID_UNREGISTER MetricsEventID = "agent.hybrid.unregister"
	EVENT_SESSION_FAILED    MetricsEventID = "agent.session.failed"
	EVENT_PERF_CPU_OVERLOAD MetricsEventID = "agent.pref.cup.overload"
	EVENT_PERF_MEM_OVERLOAD MetricsEventID = "agent.pref.mem.overload"
	EVENT_BASE_STARTUP      MetricsEventID = "agent.startup"
	EVENT_BASE_VIRTIO       MetricsEventID = "agent.virtio"
	EVENT_KDUMP             MetricsEventID = "agent.kdump"
	EVENT_PLUGIN_EXECUTE    MetricsEventID = "agent.plugin.execute"
	EVENT_PLUGIN_LOCALLIST  MetricsEventID = "agent.plugin.locallist"
	EVENT_PLUGIN_UPDATE     MetricsEventID = "agent.plugin.update"

	// event category
	EVENT_CATEGORY_CHANNEL EventCategory = "CHANNEL"
	EVENT_CATEGORY_UPDATE  EventCategory = "UPDATE"
	EVENT_CATEGORY_TASK    EventCategory = "TASK"
	EVENT_CATEGORY_HYBRID  EventCategory = "HYBRID"
	EVENT_CATEGORY_SESSION EventCategory = "SESSION"
	EVENT_CATEGORY_PERF    EventCategory = "PERF"
	EVENT_CATEGORY_STARTUP EventCategory = "STARTUP"
	EVENT_CATEGORY_VIRTIO  EventCategory = "VIRTIO"
	EVENT_CATEGORY_KDUMP   EventCategory = "KDUMP"
	EVENT_CATEGORY_PLUGIN  EventCategory = "PLUGIN"

	// event subcategory
	EVENT_SUBCATEGORY_CHANNEL_GSHELL    EventSubCategory = "gshell"
	EVENT_SUBCATEGORY_CHANNEL_WS        EventSubCategory = "ws"
	EVENT_SUBCATEGORY_CHANNEL_MGR       EventSubCategory = "channelmgr"
	EVENT_SUBCATEGORY_HYBRID_REGISTER   EventSubCategory = "register"
	EVENT_SUBCATEGORY_HYBRID_UNREGISTER EventSubCategory = "unregister"
	EVENT_SUBCATEGORY_PERF_CPU          EventSubCategory = "cpu"
	EVENT_SUBCATEGORY_PERF_MEM          EventSubCategory = "mem"

	// event level
	EVENT_LEVEL_ERROR EventLevel = "ERROR"
	EVENT_LEVEL_WARN EventLevel = "WARN"
	EVENT_LEVEL_INFO  EventLevel = "INFO"
)

type MetricsEvent struct {
	EventId     MetricsEventID   `json:"eventId"`
	Category    EventCategory    `json:"category"`
	SubCategory EventSubCategory `json:"subCategory"`
	EventLevel  EventLevel       `json:"eventLevel"`
	EventTime   int64            `json:"eventTime"`
	Common      string           `json:"common"`
	KeyWords    string           `json:"keywords"`
}

// 作为延迟上报的缓冲区
type ReportBuff struct {
	ReportChan chan string
}

type CommonInfo struct {
	Arch          string `json:"arch"`
	InstanceId    string `json:"instanceId"`
	OsVersion     string `json:"osVersion"`
	VirtualType   string `json:"virtualType"`
	Distribution  string `json:"distribution"`
	KernelVersion string `json:"kernekVersion"`
}

func (m *MetricsEvent) ReportEvent() {
	// 序列化后上报
	payload, err := json.Marshal(m)
	if err != nil {
		log.GetLogger().Errorf("metrics json.Marshal err: %s", err.Error())
		return
	}
	url := util.GetMetricsService()
	go func() {
		doReport(url, string(payload))
	}()
}

// 同步上报事件，acs-plugin-manager上报时使用该方法
func (m *MetricsEvent) ReportEventSync() {
	payload, err := json.Marshal(m)
	if err != nil {
		log.GetLogger().Errorf("metrics json.Marshal err: %s", err.Error())
		return
	}
	url := util.GetMetricsService()
	util.HttpPost(url, string(payload), "")
}

func doReport(url, payload string) {
	_reportMutex.Lock()
	defer _reportMutex.Unlock()
	if _reportCounter >= _reportCounterLimit {
		gap := time.Since(_startTime)
		if gap.Minutes() > 10 {
			_reportCounter = 1
			_startTime = time.Now()
			util.HttpPost(url, payload, "")
		} else {
			return
		}
	} else {
		_reportCounter++
		util.HttpPost(url, payload, "")
	}
}

func genKeyWordsStr(keywords ...string) string {
	if len(keywords) >= 2 {
		kmp := make(map[string]string)
		for i := 0; i < len(keywords); i += 2 {
			kmp[keywords[i]] = keywords[i+1]
		}
		kmpStr, _ := json.Marshal(&kmp)
		return string(kmpStr)
	}
	return ""
}

func getCommonInfoStr() string {
	_initCommonInfoStrOnce.Do(func() {
		_commonInfo.Arch = osutil.GetOsArch()
		_commonInfo.OsVersion = osutil.GetVersion()
		_commonInfo.VirtualType = osutil.GetVirtualType()
		_commonInfo.InstanceId = util.GetInstanceId()
		platFormName, _ := osutil.PlatformName()
		platFormVersion, _ := osutil.PlatformVersion()
		_commonInfo.Distribution = platFormName + " " + platFormVersion
		_commonInfo.KernelVersion = getKernelVersion()
		str, err := json.Marshal(&_commonInfo)
		if err != nil {
			log.GetLogger().Errorf("metrics Marshal _commonInfo err: %s", err.Error())
		}
		_commonInfoStr = string(str)
	})
	return _commonInfoStr
}

// 通道系统
func GetChannelFailEvent(subCategory EventSubCategory, keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:     EVENT_CHANNEL_FAILED,
		Category:    EVENT_CATEGORY_CHANNEL,
		SubCategory: subCategory,
		EventLevel:  EVENT_LEVEL_ERROR,
		EventTime:   time.Now().UnixNano() / 1e6,
		Common:      getCommonInfoStr(),
		KeyWords:    genKeyWordsStr(keywords...),
	}
	return event
}
func GetChannelSwitchEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_CHANNEL_SWITCH,
		Category:   EVENT_CATEGORY_CHANNEL,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

// 升级系统
func GetUpdateFailedEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_UPDATE_FAILED,
		Category:   EVENT_CATEGORY_UPDATE,
		EventLevel: EVENT_LEVEL_ERROR,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

// task系统
func GetTaskFailedEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_TASK_FAILED,
		Category:   EVENT_CATEGORY_TASK,
		EventLevel: EVENT_LEVEL_ERROR,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}
func GetTaskWarnEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_TASK_WARN,
		Category:   EVENT_CATEGORY_TASK,
		EventLevel: EVENT_LEVEL_WARN,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

// 混合云系统
func GetHybridRegisterEvent(success bool, keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:     EVENT_HYBRID_REGISTER,
		Category:    EVENT_CATEGORY_HYBRID,
		SubCategory: EVENT_SUBCATEGORY_HYBRID_REGISTER,
		EventTime:   time.Now().UnixNano() / 1e6,
		Common:      getCommonInfoStr(),
		KeyWords:    genKeyWordsStr(keywords...),
	}
	if success {
		event.EventLevel = EVENT_LEVEL_INFO
	} else {
		event.EventLevel = EVENT_LEVEL_ERROR
	}
	return event
}

func GetHybridUnregisterEvent(success bool, keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:     EVENT_HYBRID_UNREGISTER,
		Category:    EVENT_CATEGORY_HYBRID,
		SubCategory: EVENT_SUBCATEGORY_HYBRID_UNREGISTER,
		EventTime:   time.Now().UnixNano() / 1e6,
		Common:      getCommonInfoStr(),
		KeyWords:    genKeyWordsStr(keywords...),
	}
	if success {
		event.EventLevel = EVENT_LEVEL_INFO
	} else {
		event.EventLevel = EVENT_LEVEL_ERROR
	}
	return event
}

// session manager系统
func GetSessionFailedEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_SESSION_FAILED,
		Category:   EVENT_CATEGORY_SESSION,
		EventLevel: EVENT_LEVEL_ERROR,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

func GetVirtioVersionEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_BASE_VIRTIO,
		Category:   EVENT_CATEGORY_VIRTIO,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

// 性能上报
func GetCpuOverloadEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:     EVENT_PERF_CPU_OVERLOAD,
		Category:    EVENT_CATEGORY_PERF,
		SubCategory: EVENT_SUBCATEGORY_PERF_CPU,
		EventLevel:  EVENT_LEVEL_ERROR,
		EventTime:   time.Now().UnixNano() / 1e6,
		Common:      getCommonInfoStr(),
		KeyWords:    genKeyWordsStr(keywords...),
	}
	return event
}

func GetMemOverloadEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:     EVENT_PERF_MEM_OVERLOAD,
		Category:    EVENT_CATEGORY_PERF,
		SubCategory: EVENT_SUBCATEGORY_PERF_MEM,
		EventLevel:  EVENT_LEVEL_ERROR,
		EventTime:   time.Now().UnixNano() / 1e6,
		Common:      getCommonInfoStr(),
		KeyWords:    genKeyWordsStr(keywords...),
	}
	return event
}

// 基础事件
func GetBaseStartupEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_BASE_STARTUP,
		Category:   EVENT_CATEGORY_STARTUP,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

// ecs_dump服务状态上报
func GetKdumpServiceStatusEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_KDUMP,
		Category:   EVENT_CATEGORY_KDUMP,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

func GetPluginExecuteEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_PLUGIN_EXECUTE,
		Category:   EVENT_CATEGORY_PLUGIN,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

func GetPluginLocalListEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_PLUGIN_LOCALLIST,
		Category:   EVENT_CATEGORY_PLUGIN,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}

func GetPluginUpdateEvent(keywords ...string) *MetricsEvent {
	event := &MetricsEvent{
		EventId:    EVENT_PLUGIN_UPDATE,
		Category:   EVENT_CATEGORY_PLUGIN,
		EventLevel: EVENT_LEVEL_INFO,
		EventTime:  time.Now().UnixNano() / 1e6,
		Common:     getCommonInfoStr(),
		KeyWords:   genKeyWordsStr(keywords...),
	}
	return event
}