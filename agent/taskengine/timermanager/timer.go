package timermanager

import (
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

	next, err := t.Schedule.nextRun()
	if err != nil {
		return nil, err
	}
	wrapgo.GoWithDefaultPanicHandler(func() {
		for {
			select {
			case <- t.refreshTimer:
				; // No-op. Just start next cycle with new interval
			case <- t.skipWait:
				wrapgo.GoWithDefaultPanicHandler(func () {
					runTimer(t)
				})
			case <- t.quit:
				return
			case <- time.After(next):
				wrapgo.GoWithDefaultPanicHandler(func () {
					runTimer(t)
				})
			}
			next, _ = t.Schedule.nextRun()
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
