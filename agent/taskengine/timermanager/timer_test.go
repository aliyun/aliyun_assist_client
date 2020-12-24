package timermanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewTimer(t *testing.T) {

}

func TestIsRunning(t *testing.T) {
	timer := NewTimer(nil, nil)
	assert.Exactly(t, timer.isRunning, timer.IsRunning(),
		"IsRunning should return consistent running state from isRunning indicator")
}

func TestRefreshTimer(t *testing.T) {
	timer := NewTimer(nil, nil)
	assert.NotPanics(t, func () {
		timer.RefreshTimer()
	}, "RefreshTimer should not panic")
	assert.True(t, <-timer.refreshTimer,
		"RefreshTimer should send true through refreshTimer channel")
}

func TestSkipWaiting(t *testing.T) {
	timer := NewTimer(nil, nil)
	assert.NotPanics(t, func () {
		timer.SkipWaiting()
	}, "SkipWaiting should not panic")
	assert.True(t, <-timer.skipWait,
		"SkipWaiting should send true through skipWait channel")
}

func TestTimerStop(t *testing.T) {
	timer := NewTimer(nil, nil)
	assert.NotPanics(t, func () {
		timer.Stop()
	}, "Stop should not panic")
	assert.True(t, <-timer.quit,
		"Stop should send true through quit channel")
}

func TestSetRunning(t *testing.T) {
	timer := NewTimer(nil, nil)
	assert.NotPanics(t, func () {
		timer.setRunning(true)
	}, "setRunning should not panic")
	assert.True(t, timer.isRunning, "setRunning(true) should set isRunning indicator as true")

	assert.NotPanics(t, func () {
		timer.setRunning(false)
	}, "setRunning should not panic")
	assert.False(t, timer.isRunning, "setRunning(false) should set isRunning indicator as true")
}

func TestRunTimer(t *testing.T) {
	// TODO: Better function mock way than hand-written
	var mockFuncCalls int = 0
	var mockFunc = func() {
		mockFuncCalls++
	}

	timer := NewTimer(nil, mockFunc)
	assert.False(t, timer.isRunning, "isRunning indicator should be false initially")

	assert.NotPanics(t, func () {
		runTimer(timer)
	}, "runTimer should not panic")
	assert.Exactly(t, 1, mockFuncCalls, "Once runTimer should execute callback once and only once")
	assert.False(t, timer.isRunning, "isRunning indicator should be false after runTimer finished")
}

func TestRunTimerNoop(t *testing.T) {
	// TODO: Better function mock way than hand-written
	var mockFuncCalls int = 0
	var mockFunc = func() {
		mockFuncCalls++
	}

	timer := NewTimer(nil, mockFunc)
	timer.isRunning = true

	assert.NotPanics(t, func () {
		runTimer(timer)
	}, "runTimer should not panic")
	assert.Exactly(t, 0, mockFuncCalls,
		"runTimer should not execute callback when isRunning indicator is true")
	assert.True(t, timer.isRunning,
		"isRunning indicator should remain True after runTimer finished")
}

// TODO: Concurrent runTimer invocation, detect race condition
