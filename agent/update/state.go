package update

import (
	"sync/atomic"
)

var (
	_criticalActionRunning int32 = 0
)

// IsCriticalActionRunning returns criticalActionRunning flag as boolean variable
func IsCriticalActionRunning() bool {
	state := atomic.LoadInt32(&_criticalActionRunning)
	return state != 0
}

// setCriticalActionRunning marks criticalActionRunning flag as true
func setCriticalActionRunning() {
	atomic.StoreInt32(&_criticalActionRunning, 1)
}

// unsetCriticalActionRunning marks criticalActionRunning flag as false
func unsetCriticalActionRunning() {
	atomic.StoreInt32(&_criticalActionRunning, 0)
}
