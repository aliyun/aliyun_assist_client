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

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
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
	 processUptime int64, heartbeatCounter uint64, azoneId string, isColdstart bool) string {
	encodedOsVersion := url.QueryEscape(osVersion)
	paramChars := fmt.Sprintf("?virt_type=%s&lang=golang&os_type=%s&os_version=%s&app_version=%s&uptime=%d&timestamp=%d&pid=%d&process_uptime=%d&index=%d&az=%s",
		virtType, osType, encodedOsVersion, appVersion, uptime, timestamp, pid,
		processUptime, heartbeatCounter, azoneId)
	// Only first heart-beat need to carry cold-start flag
	if heartbeatCounter == 0 {
		paramChars = paramChars + fmt.Sprintf("&cold_start=%t", isColdstart)
	}
	url := util.GetPingService() + paramChars
	return url
}

func invokePingRequest(requestURL string) (string, error) {
	err, response := util.HttpGet(requestURL)
	if err != nil {
		return "", err
	}

	return response, nil
}

func doPing() error {
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
	isColdstart := false
	// Only first heart-beat need to carry cold-start flag
	if heartbeatCounter == 0 {
		if _isColdstart, err := flagging.IsColdstart(); err != nil {
			log.GetLogger().WithError(err).Errorln("Error encountered when detecting cold-start flag")
		} else {
			isColdstart = _isColdstart
		}
	}

	url := buildPingRequest(virtType, osType, osVersion, appVersion, startTime,
		timestamp, pid, processUptime, heartbeatCounter, azoneId, isColdstart)

	nextIntervalSeconds := DefaultPingIntervalSeconds
	newTasks := false

	res, err := invokePingRequest(url)
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
		}).WithError(err).Errorln("Failed to invoke ping request")
		// task_engine::DebugTask task;
		// task.RunSystemNetCheck();
		return err
	}

	if !gjson.Valid(res) {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("Invalid json response")
		return nil
	}

	json := gjson.Parse(res)
	nextIntervalField := json.Get("nextInterval")
	if !nextIntervalField.Exists() {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("nextInterval field not found in json response")
		return nil
	}
	nextIntervalMilliseconds, ok := nextIntervalField.Value().(float64)
	if !ok {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("Invalid nextInterval value in json response")
		return nil
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
		return nil
	}
	newTasks, ok = newTasksField.Value().(bool)
	if !ok {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL": url,
			"response": res,
		}).Errorln("Invalid newTasks value in json response")
		return nil
	}

	mutableSchedule, ok := _heartbeatTimer.Schedule.(*timermanager.MutableScheduled)
	if !ok {
		log.GetLogger().Errorln("Unexpected schedule type of heartbeat timer")
		return nil
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

	return nil
}

func PingwithRetries(retryCount int) {
	for i := 0; i < retryCount; i++ {
		if err := doPing(); err == nil {
			break
		}
	}
}

func pingWithoutRetry() {
	// Error(s) encountered during heart-beating has been logged internally,
	// simply ignore it here.
	doPing()
}

func InitHeartbeatTimer() error {
	if _heartbeatTimer == nil {
		_heartbeatTimerInitLock.Lock()
		defer _heartbeatTimerInitLock.Unlock()

		if _heartbeatTimer == nil {
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(pingWithoutRetry, DefaultPingIntervalSeconds)
			if err != nil {
				return err
			}
			_heartbeatTimer = timer

			// Heart-beat at starting SHOULD be executed in main goroutine,
			// subsequent sending would be invoked in TimerManager goroutines
			mutableSchedule, ok := _heartbeatTimer.Schedule.(*timermanager.MutableScheduled)
			if !ok {
				return errors.New("Unexpected schedule type of heart-beat timer")
			}
			mutableSchedule.NotImmediately()

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
