package timermanager

import (
	"sync"
	"time"
)

type TimerManager struct {
	timers map[*Timer]struct{}

	lock sync.Mutex
}


var (
	// Pointer to singleton object
	_timerManager *TimerManager
	// Lock for singleton object
	_timerManagerInitLock sync.Mutex
)

// InitTimerManager explicitly initialize global TimerManager instance, which
// should be invoked before all timer creation behaviors.
func InitTimerManager() error {
	if _timerManager == nil {
		_timerManagerInitLock.Lock()
		defer _timerManagerInitLock.Unlock()

		if _timerManager == nil {
			_timerManager = &TimerManager{timers: make(map[*Timer]struct{})}
		}
	}
	return nil
}

// GetTimerManager just returns pointer to global TimerManager instance. Do not
// forget to check whether it is nil before using.
func GetTimerManager() *TimerManager {
	return _timerManager
}

func (m *TimerManager) Start() {
	// TimerManager should be the centralized manager for scheduled jobs, but
	// needs better design
	return
}

func (m *TimerManager) Stop() {
	for t := range m.timers {
		t.Stop()
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	for t := range m.timers {
		delete(m.timers, t)
	}
}

func (m *TimerManager) CreateCronTimer(callback TimerCallback, cronat string) (*Timer, error) {
	s, err := NewCronScheduled(cronat)
	if err != nil {
		return nil, err
	}
	t := NewTimer(s, callback)

	m.lock.Lock()
	defer m.lock.Unlock()
	m.timers[t] = struct{}{}
	return t, nil
}

func (m *TimerManager) CreateRateTimer(callback TimerCallback, cronat string, creationTime time.Time) (*Timer, error) {
	s, err := NewRateScheduled(cronat, creationTime)
	if err != nil {
		return nil, err
	}
	t := NewTimer(s, callback)

	m.lock.Lock()
	defer m.lock.Unlock()
	m.timers[t] = struct{}{}
	return t, nil
}

func (m *TimerManager) CreateAtTimer(callback TimerCallback, cronat string) (*Timer, error) {
	s, err := NewAtScheduled(cronat)
	if err != nil {
		return nil, err
	}
	t := NewTimer(s, callback)

	m.lock.Lock()
	defer m.lock.Unlock()
	m.timers[t] = struct{}{}
	return t, nil
}

// CreateTimerInSeconds returns new registered timer in precision of seconds
func (m *TimerManager) CreateTimerInSeconds(callback TimerCallback, seconds int) (*Timer, error) {
	return m.CreateTimerInNanoseconds(callback, time.Duration(seconds) * time.Second)
}

// CreateTimerInNanoseconds returns new registered timer in precision of nanoseconds
func (m *TimerManager) CreateTimerInNanoseconds(callback TimerCallback, interval time.Duration) (*Timer, error) {
	s := NewMutableScheduled(interval)
	t := NewTimer(s, callback)

	m.lock.Lock()
	defer m.lock.Unlock()
	m.timers[t] = struct{}{}
	return t, nil
}

func (m *TimerManager) DeleteTimer(t *Timer) {
	t.Stop()

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.timers, t)
}

