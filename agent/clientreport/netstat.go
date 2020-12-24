package clientreport

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
)

const (
	_netstatReportType = "AgentNetworkConnectionStatistics"
	_netstatReportIntervalSeconds = 15 * 60
)

var (
	_netstatTimer *timermanager.Timer
	_netstatTimerInitLock sync.Mutex
	errNetstatTimerInitialized = fmt.Errorf("%s timer has been initialized", _netstatReportType)
)

func InitNetstatTimer() error {
	if _netstatTimer == nil {
		_netstatTimerInitLock.Lock()
		defer _netstatTimerInitLock.Unlock()

		if _netstatTimer == nil {
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(func() {
				_, err := ReportCommandOutput(_netstatReportType, "/bin/sh", []string{"-c", "netstat -tnp | grep aliyun"})
				if err != nil {
					log.GetLogger().WithError(err).Errorf("Failed to report %s", _netstatReportType)
				}
			}, _netstatReportIntervalSeconds)
			if err != nil {
				return err
			}
			_netstatTimer = timer

			// Due to throtting policy of client_report API, netstat job should not be exeucted immediately
			mutableSchedule, ok := _netstatTimer.Schedule.(*timermanager.MutableScheduled)
			if !ok {
				return errors.New("Unexpected schedule type of netstat timer")
			}
			mutableSchedule.NotImmediately()

			_, err = _netstatTimer.Run()
			if err != nil {
				return err
			}

			// Inititate another goroutine to sleep a while and issue first invocation
			go func() {
				time.Sleep(40 * time.Second)
				_netstatTimer.SkipWaiting()
			}()

			return nil
		}
		return errNetstatTimerInitialized
	}
	return errNetstatTimerInitialized
}
