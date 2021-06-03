package statemanager

import (
	"fmt"
	"math/rand"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
)

const (
	// Cron schedule type, expression format is defined in https://help.aliyun.com/document_detail/169784.html
	Cron = "cron"
	// Rate schedule type, expression format examples: "5 minutes" "1 hour"
	Rate = "rate"

	// Cron is triggered with a drift, after the expected time
	// This helps to reduce concurrency
	MaxCronDriftSeconds = 15 * 60
)

type StateConfigTimer struct {
	timer              *timermanager.Timer
	scheduleType       string
	scheduleExpression string
}

var (
	stateConfigTimers     = map[string]*StateConfigTimer{}
	stateConfigTimersLock sync.Mutex

	stateConfigEnforceLock sync.Mutex
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GetRateInSeconds(expr string) (seconds int, err error) {
	parts := strings.Split(expr, " ")
	if len(parts) != 2 {
		err = fmt.Errorf("wrong rate expression: %s", expr)
		return
	}
	seconds, err = strconv.Atoi(parts[0])
	if err != nil {
		err = fmt.Errorf("wrong rate expression: %s", expr)
		return
	}
	unit := parts[1]
	switch unit {
	case "second", "seconds":
		seconds *= 1
	case "minute", "minutes":
		seconds *= 60
	case "hour", "hours":
		seconds *= 60 * 60
	case "day", "days":
		seconds *= 60 * 60 * 24
	default:
		err = fmt.Errorf("wrong unit in rate expression: %s", expr)
		return
	}
	return
}

func isScheduleChanged(config StateConfiguration) bool {
	stateConfigTimer, ok := stateConfigTimers[config.StateConfigurationId]
	if !ok {
		return true
	}
	if config.ScheduleType != stateConfigTimer.scheduleType || config.ScheduleExpression != stateConfigTimer.scheduleExpression {
		return true
	}
	return false
}

func createStateConfigCallBack(stateConfigId string) timermanager.TimerCallback {
	callback := func() {
		defer func() {
			if panicPayload := recover(); panicPayload != nil {
				stacktrace := debug.Stack()
				clientreport.ReportPanic(panicPayload, stacktrace, false)
			}
		}()
		config, ok := getStateConfig(stateConfigId)
		if !ok {
			log.GetLogger().Warnf("state configuration %s does not exist", stateConfigId)
			return
		}
		if config.ScheduleType == "cron" {
			driftMills := rand.Intn(MaxCronDriftSeconds * 1000)
			log.GetLogger().Infof("delay %d milliseconds for cron to reduce concurrency", driftMills)
			time.Sleep(time.Duration(driftMills) * time.Millisecond)
		}
		stateConfigEnforceLock.Lock()
		defer stateConfigEnforceLock.Unlock()
		err := enforce(config)
		if err != nil {
			log.GetLogger().WithError(err).Errorf("enforce state configuration %s fail", config.StateConfigurationId)
		}
		runtime.GC()
	}
	return callback
}

func setupStateConfigTimer(config StateConfiguration) (err error) {
	timerManager := timermanager.GetTimerManager()
	var timer *timermanager.Timer
	if config.ScheduleType == Rate {
		var intervalSeconds int
		intervalSeconds, err = GetRateInSeconds(config.ScheduleExpression)
		if err != nil {
			return
		}
		timer, err = timerManager.CreateTimerInSeconds(createStateConfigCallBack(config.StateConfigurationId), intervalSeconds)
	} else if config.ScheduleType == Cron {
		timer, err = timerManager.CreateCronTimer(createStateConfigCallBack(config.StateConfigurationId), config.ScheduleExpression)
	} else {
		err = fmt.Errorf("Invalid schedule type %s", config.ScheduleType)
	}
	if err != nil {
		return
	}
	_, err = timer.Run()
	stateConfgTimer := StateConfigTimer{timer, config.ScheduleType, config.ScheduleExpression}
	stateConfigTimers[config.StateConfigurationId] = &stateConfgTimer
	log.GetLogger().Infof("setup timer for %s", config.StateConfigurationId)
	return
}

func tearDownStateConfigTimer(stateConfigId string) (err error) {
	stateConfgTimer, ok := stateConfigTimers[stateConfigId]
	if !ok {
		return
	}
	timermanager.GetTimerManager().DeleteTimer(stateConfgTimer.timer)
	delete(stateConfigTimers, stateConfigId)
	log.GetLogger().Infof("tear down timer for %s", stateConfigId)
	return
}

func refreshStateConfigTimers(configs []StateConfiguration) (err error) {
	stateConfigTimersLock.Lock()
	defer stateConfigTimersLock.Unlock()
	var stateConfigIds = make([]string, len(configs))
	for _, config := range configs {
		stateConfigIds = append(stateConfigIds, config.StateConfigurationId)
		refreshStateConfigTimer(config)
	}
	cleanupDeleted(stateConfigIds)
	return
}

func refreshStateConfigTimer(config StateConfiguration) (err error) {
	if isScheduleChanged(config) {
		log.GetLogger().Infof("%s schedule changed, refresh timer", config.StateConfigurationId)
		err = tearDownStateConfigTimer(config.StateConfigurationId)
		if err != nil {
			log.GetLogger().WithError(err).Error("tear down timer failed")
		}
		setupStateConfigTimer(config)
		if err != nil {
			log.GetLogger().WithError(err).Error("set up timer failed")
		}
	} else {
		log.GetLogger().Debugf("%s schedule not changed", config.StateConfigurationId)
	}
	return
}

func cleanupDeleted(existStateConfigIds []string) {
	for stateConfigId := range stateConfigTimers {
		var exist = false
		for _, existId := range existStateConfigIds {
			if existId == stateConfigId {
				exist = true
				break
			}
		}
		if !exist {
			tearDownStateConfigTimer(stateConfigId)
		}
	}
}

func IsStateConfigTimerRunning() bool {
	stateConfigTimersLock.Lock()
	defer stateConfigTimersLock.Unlock()
	if stateConfigTimers != nil {
		for _, t := range stateConfigTimers {
			if t.timer.IsRunning() {
				return true
			}
		}
	}
	return false
}
