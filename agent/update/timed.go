package update

import (
	"errors"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
)

const (
	// DefaultCheckIntervalSeconds is the default interval for update check timer
	DefaultCheckIntervalSeconds = 1800
)

var (
	// TODO: Centralized manager for timers of essential tasks
	_checkTimer *timermanager.Timer
	// TODO: Centralized manager for timers of essential tasks, then get rid of this
	_checkTimerInitLock sync.Mutex
)

func doCheck() {
	if err := SafeUpdate(); err != nil {
		log.GetLogger().WithError(err).Errorln("Failed to check update periodically")
	}
}

func InitCheckUpdateTimer() error {
	if _checkTimer == nil {
		_checkTimerInitLock.Lock()
		defer _checkTimerInitLock.Unlock()

		if _checkTimer == nil {
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(doCheck, DefaultCheckIntervalSeconds)
			if err != nil {
				return err
			}
			_checkTimer = timer

			// Checking update at starting SHOULD be executed in main goroutine,
			// subsequent checking would be invoked in TimerManager goroutines
			mutableSchedule, ok := _checkTimer.Schedule.(*timermanager.MutableScheduled)
			if !ok {
				return errors.New("Unexpected schedule type of netstat timer")
			}
			mutableSchedule.NotImmediately()

			_, err = _checkTimer.Run()
			if err != nil {
				return err
			}
			return nil
		}
		return errors.New("Update check timer has been initialized")
	}
	return errors.New("Update check timer has been initialized")
}
