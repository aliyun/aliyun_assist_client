package statemanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/instance"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/statemanager/resources"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
)

const (
	// DefaultRefreshIntervalSeconds is the default interval of refresh state configurations in seconds
	DefaultRefreshIntervalSeconds = 1800
	// MaxInitTimerDriftSeconds is the default max seconds to delay when initializing state manager timer
	MaxInitTimerDriftSeconds = 3 * 60
)

const (
	// Agent applies the configuration and does nothing further
	// unless the configuration (template and parameters) is updated.
	// After initial application of a new configuration,
	// agent does not check for drift from a previously configured state.
	// Agent will attempt to apply the configuration until it is successful before ApplyOnly takes effect.
	ApplyOnly = "ApplyOnly"
	// Agent applies any new configurations.
	// After initial application of a new configuration, if the instance drifts from the desired state,
	// reports the discrepancy to server.
	// Agent will attempt to apply the configuration until it is successful before ApplyAndMonitor takes effect.
	ApplyAndMonitor = "ApplyAndMonitor"
	// Agent applies any new configurations. After initial application of a new configuration,
	// if the instance drifts from the desired state, reports the discrepancy to server,
	// and then re-applies the current configuration.
	ApplyAndAutoCorrect = "ApplyAndAutoCorrect"
)

const (
	Apply   = "Apply"
	Monitor = "Monitor"
	Skip    = "Skip"
)

const (
	Compliant    = "Compliant"
	NotCompliant = "NotCompliant"
	Failed       = "Failed"
)

var (
	_stateManageTimer         *timermanager.Timer
	_statemanageTimerInitLock sync.Mutex
	refreshIntervalSeconds    = DefaultRefreshIntervalSeconds
	_stateConfigsLock         sync.RWMutex
	stateConfigs              map[string]StateConfiguration
)

func updateStateConfigs(configs []StateConfiguration) {
	_stateConfigsLock.Lock()
	defer _stateConfigsLock.Unlock()
	stateConfigs = map[string]StateConfiguration{}
	for _, config := range configs {
		stateConfigs[config.StateConfigurationId] = config
	}
}

func getStateConfig(stateConfigId string) (config StateConfiguration, ok bool) {
	_stateConfigsLock.RLock()
	defer _stateConfigsLock.RUnlock()
	config, ok = stateConfigs[stateConfigId]
	return
}

// refreshStateConfigs pulls state configurations from server and refresh state configuration timers
func refreshStateConfigs() {
	defer func() {
		if panicPayload := recover(); panicPayload != nil {
			stacktrace := debug.Stack()
			clientreport.ReportPanic(panicPayload, stacktrace, false)
		}
	}()
	log.GetLogger().Info("refresh state configurations")
	cachedResult, err := LoadConfigCache()
	var lastCheckpoint string
	var lastCheckTime time.Time
	if err != nil {
		log.GetLogger().WithError(err).Error("load local state configuration fail")
	}
	if cachedResult != nil {
		lastCheckpoint = cachedResult.Checkpoint
		log.GetLogger().Debugf("last state configuration checkpoint: %s", lastCheckpoint)
		lastCheckTime, err = timetool.ParseApiTime(lastCheckpoint)
	} else {
		lastCheckpoint = ""
	}

	result := cachedResult
	// 如果是刚刚拉取过则使用缓存
	if time.Now().Sub(lastCheckTime).Minutes() > 1 {
		info, err := instance.GetInstanceInfo()
		log.GetLogger().Debugf("instance information: %s", info)
		if err != nil {
			log.GetLogger().WithError(err).Error("get instance info failed")
			return
		}
		resp, err := ListInstanceStateConfigurations(lastCheckpoint, info.AgentName, info.AgentVersion,
			info.ComputerName, info.PlatformName, info.PlatformType, info.PlatformVersion,
			info.IpAddress, info.RamRole)
		if err != nil {
			if resp != nil && resp.ErrCode == "ServiceNotSupported" {
				log.GetLogger().Warn("state manager feature is not supported in current region")
				oneDay := 24 * 60 * 60
				if refreshIntervalSeconds != oneDay {
					refreshIntervalSeconds = oneDay
					CancelStateManagerTimer()
					InitStateManagerTimer()
				}
				return
			}
			log.GetLogger().WithError(err).Error("fail to list state configurations")
			return
		}
		if resp.Result != nil && resp.Result.Changed {
			// 未变更的情况下，服务端没有完整返回配置，使用缓存
			result = resp.Result
			WriteConfigCache(result)
		}
	}
	var targetInterval = result.Interval
	if targetInterval == 0 {
		// interval can recover to default once api call is successful and no interval is returned from server
		targetInterval = DefaultRefreshIntervalSeconds
	}
	if targetInterval != refreshIntervalSeconds {
		// 拉取配置的间隔变更，立即重新调度，会触发再次执行本函数,本次执行可以直接返回
		log.GetLogger().Infof("state manager refresh interval changes from %d to %d seconds", refreshIntervalSeconds, targetInterval)
		refreshIntervalSeconds = targetInterval
		CancelStateManagerTimer()
		InitStateManagerTimer()
		return
	}
	log.GetLogger().Infof("use state configurations: %v", result)
	updateStateConfigs(result.StateConfigurations)
	refreshStateConfigTimers(result.StateConfigurations)
	runtime.GC()
}

func enforce(config StateConfiguration) (err error) {
	var msg string
	var mode = getMode(config)
	log.GetLogger().WithFields(logrus.Fields{
		"stateConfigurationId": config.StateConfigurationId,
		"configureMode":        mode,
	}).Infof("start enforcing state configuration")
	if mode == Skip {
		return
	}
	content, err := LoadTemplateCache(config.TemplateName, config.TemplateVersion)
	if err != nil {
		log.GetLogger().WithError(err).Warn("load template from cache failed")
	}
	if content == nil {
		resp, err2 := GetTemplate(config.TemplateName, config.TemplateVersion)
		if err2 != nil {
			log.GetLogger().WithError(err2).Error("GetTemplate failed")
			msg = fmt.Sprintf("GetTemplate %s %s failed: %s", config.TemplateName, config.TemplateVersion, err2.Error())
			reportResult(config, Failed, mode, map[string]interface{}{"message": msg})
			return err2
		} else {
			content = []byte(resp.Result.Content)
			WriteTemplateCache(config.TemplateName, config.TemplateVersion, content)
		}
	}
	resourceStates, err := ParseResourceState(content, config.Parameters)
	if err != nil {
		return
	}
	if len(resourceStates) == 0 {
		log.GetLogger().Errorf("no state definition is parsed from configuration %s", config.StateConfigurationId)
		return
	}
	var resultStatus, singleStatus string
	var extraInfo string
	var notSuccessItems = make(map[string]interface{})
	switch mode {
	case Apply:
		for index, rs := range resourceStates {
			resultStatus, extraInfo, err = rs.Apply()
			if resultStatus == Failed {
				notSuccessItems[fmt.Sprintf("%dth state", index)] = err.Error()
				break
			}
			if resultStatus == NotCompliant {
				// apply should not return NotCompliant
				resultStatus = Failed
				notSuccessItems[fmt.Sprintf("%dth state", index)] = extraInfo
				break
			}
		}
	case Monitor:
		resultStatus = Compliant
		for index, rs := range resourceStates {
			singleStatus, extraInfo, err = rs.Monitor()
			if singleStatus == Failed {
				notSuccessItems[fmt.Sprintf("%dth state", index)] = err.Error()
				resultStatus = Failed
				break
			}
			if singleStatus == NotCompliant {
				notSuccessItems[fmt.Sprintf("%dth state", index)] = extraInfo
				resultStatus = NotCompliant
			}
		}
	}
	if resultStatus == "" {
		resultStatus = Failed
	}
	log.GetLogger().WithFields(logrus.Fields{
		"stateConfigurationId": config.StateConfigurationId,
		"configureMode":        mode,
	}).Infof("result status is %s", resultStatus)
	reportResult(config, resultStatus, mode, notSuccessItems)
	return
}

func hasApplied(config StateConfiguration) bool {
	if config.SuccessfulApplyTime == "" {
		return false
	}
	if config.DefinitionUpdateTime == "" {
		log.GetLogger().Errorf("state configuration %s missing DefinitionUpdateTime", config.StateConfigurationId)
		// API should always returns DefinitionUpdateTime
		// if not, assume state definition is not updated
		return true
	}
	successfulApplyTime, err2 := timetool.ParseApiTime(config.SuccessfulApplyTime)
	defUpdateTime, err1 := timetool.ParseApiTime(config.DefinitionUpdateTime)
	if err1 != nil || err2 != nil {
		log.GetLogger().Errorf("invalid DefinitionUpdateTime %s or SuccessfulApplyTime%s", config.DefinitionUpdateTime, config.SuccessfulApplyTime)
		return true
	}
	if defUpdateTime.After(successfulApplyTime) {
		// state definition (template + parameters) updated after last successful apply
		// it means new definition has never applied once
		return false
	}
	return true
}

func getMode(config StateConfiguration) string {
	switch config.ConfigureMode {
	case ApplyOnly:
		if hasApplied(config) {
			return Skip
		}
		return Apply
	case ApplyAndMonitor:
		if hasApplied(config) {
			return Monitor
		}
		return Apply
	case ApplyAndAutoCorrect:
		return Apply
	default:
		log.GetLogger().Errorf("invalid configure mode %s", config.ConfigureMode)
		return Skip
	}
}

func reportResult(config StateConfiguration, status, mode string, extraInfo map[string]interface{}) (err error) {
	var extraInfoStr string
	if extraInfo != nil && len(extraInfo) > 0 {
		data, _ := json.Marshal(extraInfo)
		extraInfoStr = string(data)
	}
	err = PutInstanceStateReport(config.StateConfigurationId, status, extraInfoStr, mode, "")
	if err != nil {
		log.GetLogger().WithError(err).Error("put state report error")
	}
	return
}

// InitStateManagerTimer starts timer for state manage feature
func InitStateManagerTimer() error {
	if _stateManageTimer == nil {
		_statemanageTimerInitLock.Lock()
		defer _statemanageTimerInitLock.Unlock()
		if _stateManageTimer == nil {
			log.GetLogger().Infof("initialize state manager timer with interval %d seconds", refreshIntervalSeconds)
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(refreshStateConfigs, refreshIntervalSeconds)
			if err != nil {
				log.GetLogger().WithError(err).Error("create state manager timer failed")
				return err
			}
			_stateManageTimer = timer
			go func() {
				// shuffle state manager task in 3 minutes
				mills := rand.Intn(MaxInitTimerDriftSeconds * 1000)
				time.Sleep(time.Duration(mills) * time.Millisecond)
				log.GetLogger().Info("run state manager timer")
				_, err = _stateManageTimer.Run()
				if err != nil {
					log.GetLogger().WithError(err).Error("run state manager timer failed")
				}
			}()
			return nil
		}
		return errors.New("state manage timer has been initialized")
	}
	return errors.New("state manage timer has been initialized")
}

func CancelStateManagerTimer() error {
	if _stateManageTimer != nil {
		log.GetLogger().Infoln("cancel state manager timer")
		_statemanageTimerInitLock.Lock()
		defer _statemanageTimerInitLock.Unlock()
		if _stateManageTimer != nil {
			timermanager.GetTimerManager().DeleteTimer(_stateManageTimer)
			_stateManageTimer = nil
		}
	}
	return nil
}

func IsStateManagerTimerRunning() bool {
	if _stateManageTimer != nil {
		_statemanageTimerInitLock.Lock()
		defer _statemanageTimerInitLock.Unlock()
		return _stateManageTimer.IsRunning()
	}
	return false
}

func NewResourceState(state StateDef) (rs resources.ResourceState, err error) {
	switch state.ResourceType {
	case "ACS:Inventory":
		rs = &resources.InventoryState{}
		rs.Load(state.Properties)
	case "ACS:File":
		rs = &resources.FileState{}
		rs.Load(state.Properties)
	default:
		log.GetLogger().Error("unsupported resource type ", state.ResourceType)
		err = fmt.Errorf("unsupported resource type %s in template", state.ResourceType)
	}
	log.GetLogger().Infof("resource state definition: %v", rs)
	return
}
