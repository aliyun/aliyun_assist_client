package heartbeat

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

const (
	// DefaultPingIntervalSeconds is the default interval of heart-beat in seconds
	DefaultPingIntervalSeconds = 1800
	// MinimumPingIntervalSeconds limits the smallest interval of heart-beat in seconds
	MinimumPingIntervalSeconds = 30
)

var (
	// TODO: Centralized manager for timers of essential tasks
	_heartbeatTimer *timermanager.Timer
	// TODO: Centralized manager for timers of essential tasks, then get rid of this
	_heartbeatTimerInitLock sync.Mutex

	_processStartTime int64
	_heartbeatCounter uint64
)

func init() {
	_processStartTime = timetool.GetAccurateTime()
	_heartbeatCounter = 0
}

func buildPingRequest(virtType string, osType string, osVersion string,
	 appVersion string, uptime uint64, timestamp int64, pid int,
	 processUptime int64, heartbeatCounter uint64, azoneId string) string {
	encodedOsVersion := url.QueryEscape(osVersion)
	paramChars := fmt.Sprintf("?virt_type=%s&lang=golang&os_type=%s&os_version=%s&app_version=%s&uptime=%d&timestamp=%d&pid=%d&process_uptime=%d&index=%d&az=%s",
		virtType, osType, encodedOsVersion, appVersion, uptime, timestamp, pid,
		processUptime, heartbeatCounter, azoneId)
	url := util.GetPingService() + paramChars
	return url
}

func invokePingRequest(requestURL string) (string, error) {
	err, response := util.HttpGet(requestURL)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": requestURL,
		}).Errorln("Network is unavailable for heart-beat request")
		return "", err
	}

	return response, nil
}

func doPing() {
	virtType := "kvm" // osutil.GetVirtualType() is currently unavailable
	osType := osutil.GetOsType()
	osVersion := osutil.GetVersion()
	appVersion := version.AssistVersion
	startTime := osutil.GetUptimeOfMs()
	timestamp := timetool.GetAccurateTime()
	pid := os.Getpid()
	processUptime := timetool.GetAccurateTime() - _processStartTime
	heartbeatCounter := _heartbeatCounter
	azoneId := util.GetAzoneId()

	url := buildPingRequest(virtType, osType, osVersion, appVersion, startTime,
		timestamp, pid, processUptime, heartbeatCounter, azoneId)

	nextIntervalSeconds := DefaultPingIntervalSeconds
	newTasks := false

	res, err := invokePingRequest(url)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
		}).WithError(err).Errorln("Failed to invoke ping request")
		// task_engine::DebugTask task;
		// task.RunSystemNetCheck();
		return
	}

	if !gjson.Valid(res) {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("Invalid json response")
		return
	}

	json := gjson.Parse(res)
	nextIntervalField := json.Get("nextInterval")
	if !nextIntervalField.Exists() {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("nextInterval field not found in json response")
		return
	}
	nextIntervalMilliseconds, ok := nextIntervalField.Value().(float64)
	if !ok {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("Invalid nextInterval value in json response")
		return
	}
	nextIntervalSeconds = int(nextIntervalMilliseconds) / 1000
	if nextIntervalSeconds < MinimumPingIntervalSeconds {
		nextIntervalSeconds = MinimumPingIntervalSeconds
	}

	newTasksField := json.Get("newTasks")
	if !newTasksField.Exists() {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("newTasks field not found in json response")
		return
	}
	newTasks, ok = newTasksField.Value().(bool)
	if !ok {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("Invalid newTasks value in json response")
		return
	}

	mutableSchedule, ok := _heartbeatTimer.Schedule.(*timermanager.MutableScheduled)
	if !ok {
		log.GetLogger().Errorln("Unexpected schedule type of heartbeat timer")
		return
	}
	// Not so graceful way to reset interval of timer: too much implementation exposed.
	mutableSchedule.SetInterval(time.Duration(nextIntervalSeconds) * time.Second)
	_heartbeatTimer.RefreshTimer()

	log.GetLogger().WithFields(logrus.Fields{
		"requestURL": url,
		"response": res,
		"nextIntervalSeconds": nextIntervalSeconds,
		"newTasks": newTasks,
	}).Infoln("Ping request succeeds")
	_heartbeatCounter++;
}

func InitHeartbeatTimer() error {
	if _heartbeatTimer == nil {
		_heartbeatTimerInitLock.Lock()
		defer _heartbeatTimerInitLock.Unlock()

		if _heartbeatTimer == nil {
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(doPing, DefaultPingIntervalSeconds)
			if err != nil {
				return err
			}
			_heartbeatTimer = timer

			_, err = _heartbeatTimer.Run()
			if err != nil {
				return err
			}
			return nil
		}
		return errors.New("Heartbeat timer has been initialized")
	}
	return errors.New("Heartbeat timer has been initialized")
}
