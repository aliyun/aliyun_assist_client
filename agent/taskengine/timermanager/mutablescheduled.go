package timermanager

import (
	"errors"
	"time"
)

type MutableScheduled struct {
	interval time.Duration
	immediatelyDone bool
}

func NewMutableScheduled(interval time.Duration) *MutableScheduled {
	return &MutableScheduled{
		interval: interval,
		immediatelyDone: false,
	}
}

func (m *MutableScheduled) nextRun() (time.Duration, error) {
	if m.interval == 0 {
		return time.Duration(0), errors.New("cannot set interval time with 0")
	}
	if !m.immediatelyDone {
		m.immediatelyDone = true
		return 0, nil
	}
	return m.interval, nil
}

func (m *MutableScheduled) SetInterval(newInterval time.Duration) *MutableScheduled {
	m.interval = newInterval
	return m
}

func (m *MutableScheduled) NotImmediately() *MutableScheduled {
	m.immediatelyDone = true
	return m
}
