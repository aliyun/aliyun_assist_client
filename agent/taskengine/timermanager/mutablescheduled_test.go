package timermanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewMutableScheduled(t *testing.T) {
	var interval time.Duration = 7136 * time.Second

	mutableScheduled := NewMutableScheduled(interval)
	assert.Exactly(t, interval, mutableScheduled.interval,
		"NewMutableScheduled should copy interval setting")
	assert.Exactly(t, false, mutableScheduled.immediatelyDone,
		"NewMutableScheduled should set immediatelyDone false by default")
}

func TestNextRunError(t *testing.T) {
	var interval time.Duration = 0 * time.Second

	mutableScheduled := NewMutableScheduled(interval)
	next, err := mutableScheduled.nextRun()
	assert.Error(t, err, "NextRun should failed on case when interval is set to 0")
	assert.Exactly(t, time.Duration(0), next, "NextRun should return zero value of time.Duration when failed")
}

func TestNotImmediatelyRun(t *testing.T) {
	var interval time.Duration = 7136 * time.Second

	mutableScheduled := NewMutableScheduled(interval).NotImmediately()
	next, err := mutableScheduled.nextRun()
	assert.NoError(t, err, "NextRun should succeed when interval is valid")
	assert.Exactly(t, interval, next, "NextRun should return preset interval")
}

func TestNextRun(t *testing.T) {
	var interval time.Duration = 7136 * time.Second

	mutableScheduled := NewMutableScheduled(interval)
	firstRun, firstErr := mutableScheduled.nextRun()
	assert.NoError(t, firstErr, "NextRun should succeed when needed to run immediately")
	assert.Exactly(t, time.Duration(0), firstRun, "NextRun should return 0 as interval to run immediately")

	secondRun, secondErr := mutableScheduled.nextRun()
	assert.NoError(t, secondErr, "NextRun should succeed when normally calling nextRun")
	assert.Exactly(t, interval, secondRun, "NextRun should return preset interval to run periodically")
}

func TestSetInterval(t *testing.T) {
	var originalInterval time.Duration = 1 * time.Second
	var newInterval time.Duration = 7136 * time.Second

	mutableScheduled := NewMutableScheduled(originalInterval)
	mutableScheduled.SetInterval(newInterval)
	assert.Exactly(t, newInterval, mutableScheduled.interval,
		"SetInterval should overwrite interval setting")
}

func TestNotImmediately(t *testing.T) {
	var interval time.Duration = 7136 * time.Second

	mutableScheduled := NewMutableScheduled(interval)
	mutableScheduled.NotImmediately()
	assert.Exactly(t, true, mutableScheduled.immediatelyDone,
		"NotImmediately should set immediatelyDone true")
}
