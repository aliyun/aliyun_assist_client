package heartbeat

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"

	"github.com/aliyun/aliyun_assist_client/agent/checknet"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/agent/version"
	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/machineid"
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

	_processStartTime   time.Time
	_acknowledgeCounter uint64
	_sendCounter        uint64

	_lastHttpFailedSendCounter uint64 // record the _sendCounter value when http ping failed
	_tryHttp                   bool

	_machineId string

	_intervalRand *rand.Rand

	// if _useFullFields is true ping hear-beat with full fields,
	// otherwise use the reduced fields
	_useFullFields atomic.Bool

	_networkConnected          atomic.Bool
	_actionsWhenNetRecover     map[string]func()
	_actionsWhenNetRecoverLock sync.Mutex

	_consecutiveFailedCount int
)

func init() {
	_retryCounter = 0
	_retryMutex = &sync.Mutex{}
	_processStartTime = time.Now()
	_acknowledgeCounter = 0
	_sendCounter = 0

	_lastHttpFailedSendCounter = 0
	_tryHttp = true

	_machineId, _ = machineid.GetMachineID()

	_intervalRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

// retrievePingResponse would always return response content and error, although
// either response content may be empty, or error is nil.
//
// * Response content would be empty when network error encountered.
// * But non-empty response content would be returned along with
//   [httpbase.StatusCodeError] for further processing.
//
//   See caller [invokePingRequest] for usages.
func retrievePingResponse(isHttpScheme bool, urlWithoutScheme string, willSwitchScheme bool) (string, error) {
	httpRequestURL := "http://" + urlWithoutScheme
	httpsRequestURL := "https://" + urlWithoutScheme
	var requestURL, switchedRequestUrl *string
	if isHttpScheme {
		requestURL = &httpRequestURL
		switchedRequestUrl = &httpsRequestURL
	} else {
		requestURL = &httpsRequestURL
		switchedRequestUrl = &httpRequestURL
	}
	err, response := util.HttpGet(*requestURL)
	if err != nil {
		tmp_err, ok := err.(*httpbase.StatusCodeError)
		if !(ok && tmp_err.StatusCode() < 500) {
			_retryMutex.Lock()
			defer _retryMutex.Unlock()
			Gap := time.Since(_startTime)
			//more than 1h than reset counter and start time.
			if Gap.Minutes() >= 60 {
				_retryCounter = 0
				_startTime = time.Now()
			}
			//less than 1h and counter more than 3.
			if _retryCounter >= 3 {
				log.GetLogger().WithFields(log.Fields{
					"requestURL": *requestURL,
					"response":   response,
				}).WithError(err).Errorln("Retry too frequent")
			} else {
				//do retry
				time.Sleep(3 * time.Second)
				_retryCounter++
				err, response = util.HttpGet(*requestURL)
				if err == nil {
					// Keep use current scheme next time
					if isHttpScheme {
						_tryHttp = true
					} else {
						_tryHttp = false
						_lastHttpFailedSendCounter = _sendCounter
					}
					return response, nil
				}
				log.GetLogger().WithFields(log.Fields{
					"requestURL": *requestURL,
					"response":   response,
				}).WithError(err).Errorln("Retry failed")
				if willSwitchScheme {
					err, response = util.HttpGet(*switchedRequestUrl)
					if err == nil {
						// Use another scheme next time
						if isHttpScheme {
							_tryHttp = false
							_lastHttpFailedSendCounter = _sendCounter
						} else {
							_tryHttp = true
						}
						return response, nil
					}
					log.GetLogger().WithFields(log.Fields{
						"switchedRequestURL": *switchedRequestUrl,
						"response":           response,
					}).WithError(err).Errorln("Retry failed with switched requestURL")
				}
			}
		}
		// Use another scheme next time
		if isHttpScheme {
			_tryHttp = false
			_lastHttpFailedSendCounter = _sendCounter
		} else {
			_tryHttp = true
		}

		return response, err
	}
	return response, nil
}

// invokePingRequest would return either response content or error (e.g.,
// [httpbase.StatusCodeError], [errMissingRequiredField], or others), but NEVER
// BOTH.
//
// Possible side-effect(s): [_instanceDeregisteredHook]
func invokePingRequest(isHttpScheme bool, urlWithoutScheme string, willSwitchScheme bool) (string, error) {
	response, err := retrievePingResponse(isHttpScheme, urlWithoutScheme, willSwitchScheme)

	if errInResponse := digErrorFromResponse(response); errInResponse != nil {
		err = errInResponse
	}

	if err != nil {
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
	sendCounter := _sendCounter
	var querystring string
	if _useFullFields.Load() {
		querystring = buildFullFieldsPingParams(sendCounter)
	} else {
		querystring = buildPingParams(sendCounter)
	}

	// Use non-secure HTTP protocol by default to reduce performance impact from
	// TLS in trustable network environment...
	isHttpScheme := true
	willSwitchScheme := true
	// If HTTP protocol is not accessible use HTTPS. Actively try the http protocol
	// after 24 * 60 heart-beats
	if !_tryHttp {
		if sendCounter-_lastHttpFailedSendCounter > 24*60 {
			log.GetLogger().Info("heart-beat by https more than 24*60 times, try http")
			isHttpScheme = true
			_tryHttp = true
		} else {
			isHttpScheme = false
		}
	}
	// ...but internet for hybrid mode is obviously untrusted
	if instance.IsHybrid() {
		isHttpScheme = false
		willSwitchScheme = false
	}
	urlWithoutScheme := util.GetPingService() + querystring

	responseContent, err := invokePingRequest(isHttpScheme, urlWithoutScheme, willSwitchScheme)
	if err != nil {
		_consecutiveFailedCount += 1
		_networkConnected.Store(false)
		log.GetLogger().WithFields(log.Fields{
			"requestURLWithoutScheme": urlWithoutScheme,
			"isHttpScheme":            isHttpScheme,
		}).WithError(err).Errorln("Failed to invoke ping request")

		if _consecutiveFailedCount >= 3 {
			go checknet.ReportNetworkBlockToSerialPort(err)
		}
		return err
	} else {
		_consecutiveFailedCount = 0
		if _networkConnected.CompareAndSwap(false, true) {
			go func() {
				_actionsWhenNetRecoverLock.Lock()
				defer _actionsWhenNetRecoverLock.Unlock()
				for name, action := range _actionsWhenNetRecover {
					log.GetLogger().Infof("Execute action[%s] after network recover", name)
					action()
				}
			}()
		}
	}

	mutableSchedule, ok := _heartbeatTimer.Schedule.(*timermanager.MutableScheduled)
	if !ok {
		log.GetLogger().Errorln("Unexpected schedule type of heartbeat timer")
		return nil
	}
	// Not so graceful way to reset interval of timer: too much implementation exposed.
	mutableSchedule.SetInterval(extractNextInterval(responseContent))

	err = flagging.UpdateConfOnConfigReceived(log.GetLogger(), responseContent)
	if err != nil {
		log.GetLogger().WithFields(log.Fields{"response": responseContent}).WithError(err).Errorln("UpdateConfOnConfigReceived failed in heartbeat.")
	}
	_heartbeatTimer.RefreshTimer()

	return nil
}

// buildPingParams constructs simplified heartbeat request parameters
func buildPingParams(sendCounter uint64) (querystring string) {
	uptime := osutil.GetUptimeOfMs()
	timestamp := timetool.GetAccurateTime()
	pid := os.Getpid()
	processUptime := time.Since(_processStartTime).Milliseconds()
	acknowledgeCounter := _acknowledgeCounter
	querystring = fmt.Sprintf("?uptime=%d&timestamp=%d&pid=%d&process_uptime=%d&index=%d&seq_no=%d",
		uptime, timestamp, pid, processUptime, acknowledgeCounter, sendCounter)

	// Only first heart-beat need to carry extra params
	if acknowledgeCounter == 0 {
		isColdstart := false
		if _isColdstart, err := flagging.IsColdstart(); err != nil {
			log.GetLogger().WithError(err).Errorln("Error encountered when detecting cold-start flag")
		} else {
			isColdstart = _isColdstart
		}

		virtType := "kvm" // osutil.GetVirtualType() is currently unavailable
		osVersion := osutil.GetVersion()
		azoneId := util.GetAzoneId()

		encodedOsVersion := url.QueryEscape(osVersion)
		querystring += fmt.Sprintf("&virt_type=%s&os_version=%s&az=%s&machineid=%s&cold_start=%t", virtType,
			encodedOsVersion, azoneId, _machineId, isColdstart)
	}
	return
}

// buildFullFieldsPingParams constructs a full set of heartbeat request
// parameters to be compatible with servers that do not recognize the simplified
// heartbeat parameters.
func buildFullFieldsPingParams(sendCounter uint64) (querystring string) {
	uptime := osutil.GetUptimeOfMs()
	timestamp := timetool.GetAccurateTime()
	pid := os.Getpid()
	processUptime := time.Since(_processStartTime).Milliseconds()
	acknowledgeCounter := _acknowledgeCounter
	querystring = fmt.Sprintf("?uptime=%d&timestamp=%d&pid=%d&process_uptime=%d&index=%d&seq_no=%d",
		uptime, timestamp, pid, processUptime, acknowledgeCounter, sendCounter)

	virtType := "kvm" // osutil.GetVirtualType() is currently unavailable
	osType := osutil.GetOsType()
	osVersion := url.QueryEscape(osutil.GetVersion())
	azId := util.GetAzoneId()
	querystring += fmt.Sprintf("&virt_type=%s&lang=golang&os_type=%s&os_version=%s&app_version=%s&az=%s",
		virtType, osType, osVersion, version.AssistVersion, azId)

	// Only first heart-beat need to carry extra params
	if acknowledgeCounter == 0 {
		isColdstart := false
		if _isColdstart, err := flagging.IsColdstart(); err != nil {
			log.GetLogger().WithError(err).Errorln("Error encountered when detecting cold-start flag")
		} else {
			isColdstart = _isColdstart
		}

		querystring += fmt.Sprintf("&cold_start=%t", isColdstart)
	}
	return
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

func RegisterActionWhenNetRecover(actions map[string]func()) {
	_actionsWhenNetRecoverLock.Lock()
	defer _actionsWhenNetRecoverLock.Unlock()
	if _actionsWhenNetRecover == nil {
		_actionsWhenNetRecover = make(map[string]func())
	}
	for name, f := range actions {
		_actionsWhenNetRecover[name] = f
	}
}
