package timermanager

import (
	"errors"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/wrapgo"
)

type scheduled interface {
	nextRun() (time.Duration, error)
}

type TimerCallback func()

type Timer struct {
	Schedule scheduled
	callback TimerCallback

	refreshTimer chan bool
	skipWait chan bool
	quit chan bool

	rwLock sync.RWMutex
	isRunning bool
	err error
}

var (
	ErrNoNextRun = errors.New("Schedule has finished and no next run")
)

func NewTimer(s scheduled, c TimerCallback) *Timer {
	return &Timer{
		Schedule: s,
		callback: c,

		refreshTimer: make(chan bool, 1),
		skipWait: make(chan bool, 1),
		quit: make(chan bool, 1),

		isRunning: false,
		err: nil,
	}
}

func (t *Timer) IsRunning() bool {
	t.rwLock.RLock()
	defer t.rwLock.RUnlock()
	return t.isRunning
}

func (t *Timer) Run() (*Timer, error) {
	if t.err != nil {
		return nil, t.err
	}

	durationToWait, err := t.Schedule.nextRun()
	if err != nil {
		return nil, err
	}
	if durationToWait < 0 {
		return nil, ErrNoNextRun
	}
	wrapgo.GoWithDefaultPanicHandler(func() {
		for shouldContinue := true; shouldContinue; {
			if durationToWait < 0 {
				return
			}

			shouldContinue = func () bool {
				timer := time.NewTimer(durationToWait)
				defer timer.Stop()

				select {
				case <- t.refreshTimer:
					; // No-op. Just start next cycle with new interval
				case <- t.skipWait:
					wrapgo.GoWithDefaultPanicHandler(func () {
						runTimer(t)
					})
				case <- t.quit:
					return false
				case <- timer.C:
					wrapgo.GoWithDefaultPanicHandler(func () {
						runTimer(t)
					})
				}

				return true
			}()
			durationToWait, _ = t.Schedule.nextRun()
		}
	})
	return t, nil
}

func (t *Timer) RefreshTimer() {
	t.refreshTimer <- true
}

func (t *Timer) SkipWaiting() {
	t.skipWait <- true
}

func (t *Timer) Stop() {
	t.quit <- true
}

func (t *Timer) setRunning(state bool) {
	t.rwLock.Lock()
	defer t.rwLock.Unlock()
	t.isRunning = state
}

func runTimer(t *Timer) {
	if t.IsRunning() {
		return
	}
	t.setRunning(true)
	t.callback()
	t.setRunning(false)
}
