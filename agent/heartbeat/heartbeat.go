package heartbeat

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/common/apiserver"
	"github.com/aliyun/aliyun_assist_client/common/machineid"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	// DefaultPingIntervalSeconds is the default interval of heart-beat in seconds
	DefaultPingIntervalSeconds = 60

	leastIntervalInMilliseconds = 55000
	mostIntervalInMilliseconds  = 65000
)

var (
	// TODO: Centralized manager for timers of essential tasks
	_heartbeatTimer *timermanager.Timer
	// TODO: Centralized manager for timers of essential tasks, then get rid of this
	_heartbeatTimerInitLock sync.Mutex

	_startTime    time.Time
	_retryCounter uint16
	_retryMutex   *sync.Mutex

	_processStartTime   int64
	_acknowledgeCounter uint64
	_sendCounter        uint64

	_machineId string

	_intervalRand *rand.Rand
)

func init() {
	_retryCounter = 0
	_retryMutex = &sync.Mutex{}
	_processStartTime = timetool.GetAccurateTime()
	_acknowledgeCounter = 0
	_sendCounter = 0

	_machineId, _ = machineid.GetMachineID()

	_intervalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func buildBootstrapPingParams() string {
	virtType := "kvm" // osutil.GetVirtualType() is currently unavailable
	osVersion := osutil.GetVersion()
	azoneId := util.GetAzoneId()
	isColdstart := false
	if _isColdstart, err := flagging.IsColdstart(); err != nil {
		log.GetLogger().WithError(err).Errorln("Error encountered when detecting cold-start flag")
	} else {
		isColdstart = _isColdstart
	}

	encodedOsVersion := url.QueryEscape(osVersion)
	return fmt.Sprintf("&virt_type=%s&os_version=%s&az=%s&machineid=%s&cold_start=%t", virtType,
		encodedOsVersion, azoneId, _machineId, isColdstart)
}

func invokePingRequest(requestURL string) (string, error) {
	err, response := util.HttpGet(requestURL)
	if err != nil {
		tmp_err, ok := err.(*requester.HttpErrorCode)
		if !(ok && tmp_err.GetCode() < 500) {
			_retryMutex.Lock()
			defer _retryMutex.Unlock()
			Gap := time.Now().Sub(_startTime)
			//more than 1h than reset counter and start time.
			if Gap.Minutes() >= 60 {
				_retryCounter = 0
				_startTime = time.Now()
			}
			//less than 1h and counter more than 3.
			if _retryCounter >= 3 {
				log.GetLogger().WithFields(log.Fields{
					"requestURL": requestURL,
					"response":   response,
				}).WithError(err).Errorln("Retry too frequent")
			} else {
				//do retry
				time.Sleep(3 * time.Second)
				err, response := util.HttpGet(requestURL)
				_retryCounter++
				if err == nil {
					return response, nil
				}
				log.GetLogger().WithFields(log.Fields{
					"requestURL": requestURL,
					"response":   response,
				}).WithError(err).Errorln("Retry failed")
			}
		}
		return "", err
	}
	return response, nil
}

func randomNextInterval() time.Duration {
	nextIntervalInMilliseconds := _intervalRand.Intn(mostIntervalInMilliseconds-leastIntervalInMilliseconds+1) + leastIntervalInMilliseconds
	return time.Duration(nextIntervalInMilliseconds) * time.Millisecond
}

func extractNextInterval(content string) time.Duration {
	if !gjson.Valid(content) {
		log.GetLogger().WithFields(log.Fields{
			"response": content,
		}).Errorln("Invalid json response")
		return randomNextInterval()
	}

	json := gjson.Parse(content)
	nextIntervalField := json.Get("nextInterval")
	if !nextIntervalField.Exists() {
		log.GetLogger().WithFields(log.Fields{
			"response": content,
		}).Errorln("nextInterval field not found in json response")
		return randomNextInterval()
	}
	nextIntervalValue, ok := nextIntervalField.Value().(float64)
	if !ok {
		log.GetLogger().WithFields(log.Fields{
			"response": content,
		}).Errorln("Invalid nextInterval value in json response")
		return randomNextInterval()
	}
	nextIntervalInMilliseconds := int(nextIntervalValue)
	if nextIntervalInMilliseconds < leastIntervalInMilliseconds || nextIntervalInMilliseconds > mostIntervalInMilliseconds {
		return randomNextInterval()
	}

	return time.Duration(nextIntervalInMilliseconds) * time.Millisecond
}

func doPing() error {
	uptime := osutil.GetUptimeOfMs()
	timestamp := timetool.GetAccurateTime()
	pid := os.Getpid()
	processUptime := timetool.GetAccurateTime() - _processStartTime
	acknowledgeCounter := _acknowledgeCounter
	sendCounter := _sendCounter
	querystring := fmt.Sprintf("?uptime=%d&timestamp=%d&pid=%d&process_uptime=%d&index=%d&seq_no=%d",
		uptime, timestamp, pid, processUptime, acknowledgeCounter, sendCounter)

	// Use non-secure HTTP protocol by default to reduce performance impact from
	// TLS in trustable network environment...
	schemePart := "http://"
	// ...but internet for hybrid mode is obviously untrusted
	if apiserver.IsHybrid() {
		schemePart = "https://"
	}
	url := schemePart + util.GetPingService() + querystring
	// Only first heart-beat need to carry extra params
	if acknowledgeCounter == 0 {
		url = url + buildBootstrapPingParams()
	}

	responseContent, err := invokePingRequest(url)
	if err != nil {
		log.GetLogger().WithFields(log.Fields{
			"requestURL": url,
		}).WithError(err).Errorln("Failed to invoke ping request")
		return err
	}

	mutableSchedule, ok := _heartbeatTimer.Schedule.(*timermanager.MutableScheduled)
	if !ok {
		log.GetLogger().Errorln("Unexpected schedule type of heartbeat timer")
		return nil
	}
	// Not so graceful way to reset interval of timer: too much implementation exposed.
	mutableSchedule.SetInterval(extractNextInterval(responseContent))
	_heartbeatTimer.RefreshTimer()

	return nil
}

func PingwithRetries(retryCount int) {
	for i := 0; i < retryCount; i++ {
		if err := doPing(); err == nil {
			_acknowledgeCounter++
			break
		}
	}
	_sendCounter++
}

func pingWithoutRetry() {
	// Error(s) encountered during heart-beating has been logged internally,
	// simply ignore it here.
	if err := doPing(); err == nil {
		_acknowledgeCounter++
	}
	_sendCounter++
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
