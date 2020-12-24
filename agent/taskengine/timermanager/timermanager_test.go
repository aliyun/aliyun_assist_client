package timermanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitTimerManager(t *testing.T) {
	assert.Nil(t, _timerManager,
		"Package-private variable _timerManager should be nil before calling InitTimerManager")

	firstErr := InitTimerManager()
	assert.NoError(t, firstErr, "InitTimerManager should not return error")
	assert.NotNil(t, _timerManager,
		"InitTimerManager should set package-private variable _timerManager to non-nil pointer")

	currentTimerManager := _timerManager
	secondErr := InitTimerManager()
	assert.NoError(t, secondErr, "InitTimerManager should not return error for second call")
	assert.Exactly(t, currentTimerManager, _timerManager, "InitTimerManager should not set new value for second call and later")
}

func TestGetTimerManager(t *testing.T) {
	assert.Exactly(t, _timerManager, GetTimerManager(),
		"GetTimerManager should return same value as _timerManager")
}

func TestTimerManagerStart(t *testing.T) {
	timerManager := &TimerManager{timers: make(map[*Timer]struct{})}
	assert.NotPanics(t, func () {
		timerManager.Start()
	}, "Start should not panic")
}

func TestTimerManagerStop(t *testing.T) {
	scheduled, err := NewCronScheduled("0 */20 * * * ?")
	assert.NoError(t, err, "NewCronScheduled should not fail!")

	timer := NewTimer(scheduled, func() {})
	timerManager := &TimerManager{
		timers: map[*Timer]struct{}{
			timer: struct{}{},
		},
	}
	assert.Equal(t, 1, len(timerManager.timers),
		"Manually created TimerManager instance should contain 1 timer initially")

	assert.NotPanics(t, func () {
		timerManager.Stop()
	}, "Stop should not panic")
	assert.Equal(t, 0, len(timerManager.timers),
		"TimerManager instance should contain 0 timer after deletion")
}

func TestCreateCronTimer(t *testing.T) {
	const CronExpression = "*/20 * * * * ?"

	timerManager := &TimerManager{timers: make(map[*Timer]struct{})}
	assert.Equal(t, 0, len(timerManager.timers),
		"TimerManager instance should contain 0 timer initially")

	timer, err := timerManager.CreateCronTimer(func (){}, CronExpression)
	assert.NoErrorf(t, err,
		"CreateCronTimer should not return error for cron expression %s", CronExpression)
	if _, ok := timer.Schedule.(*CronScheduled); !ok {
		assert.FailNow(t, "CreateCronTimer should create CronScheduled for timer")
	}
	assert.Equal(t, 1, len(timerManager.timers),
		"TimerManager instance should contain 1 timer after addition")
}

func TestCreateTimerInSeconds(t *testing.T) {
	const IntervalSeconds int = 7136

	timerManager := &TimerManager{timers: make(map[*Timer]struct{})}
	assert.Equal(t, 0, len(timerManager.timers),
		"TimerManager instance should contain 0 timer initially")

	timer, err := timerManager.CreateTimerInSeconds(func() {}, IntervalSeconds)
	assert.NoErrorf(t, err,
		"CreateTimerInSeconds should not return error for %d second interval", IntervalSeconds)
	mutableScheduled, ok := timer.Schedule.(*MutableScheduled)
	if !ok {
		assert.FailNow(t, "CreateTimerInSeconds should create MutableScheduled for timer")
	}
	assert.Exactlyf(t, time.Duration(IntervalSeconds) * time.Second, mutableScheduled.interval,
		"CreateTimerInSeconds should set correct interval for %d second interval", IntervalSeconds)
	assert.Equal(t, 1, len(timerManager.timers),
		"TimerManager instance should contain 1 timer after addition")
}

func TestCreateTimerInNanoseconds(t *testing.T) {
	const IntervalNanoseconds = 7136 * time.Second

	timerManager := &TimerManager{timers: make(map[*Timer]struct{})}
	assert.Equal(t, 0, len(timerManager.timers),
		"TimerManager instance should contain 0 timer initially")

	timer, err := timerManager.CreateTimerInNanoseconds(func() {}, IntervalNanoseconds)
	assert.NoErrorf(t, err,
		"CreateTimerInNanoseconds should not return error for %d nanosecond interval", IntervalNanoseconds)
	mutableScheduled, ok := timer.Schedule.(*MutableScheduled)
	if !ok {
		assert.FailNow(t, "CreateTimerInNanoseconds should create MutableScheduled for timer")
	}
	assert.Exactlyf(t, IntervalNanoseconds, mutableScheduled.interval,
		"CreateTimerInNanoseconds should set correct interval for %d nanosecond interval", IntervalNanoseconds)
	assert.Equal(t, 1, len(timerManager.timers),
		"TimerManager instance should contain 1 timer after addition")
}

func TestDeleteTimer(t *testing.T) {
	scheduled, err := NewCronScheduled("0 */20 * * * ?")
	assert.NoError(t, err, "NewCronScheduled should not fail!")

	timer := NewTimer(scheduled, func() {})
	timerManager := &TimerManager{
		timers: map[*Timer]struct{}{
			timer: struct{}{},
		},
	}
	assert.Equal(t, 1, len(timerManager.timers),
		"Manually created TimerManager instance should contain 1 timer initially")

	timerManager.DeleteTimer(timer)
	assert.Equal(t, 0, len(timerManager.timers),
		"TimerManager instance should contain 0 timer after deletion")
}
