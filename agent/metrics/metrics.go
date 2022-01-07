package metrics

import (
	"sync"
	"time"
)
var (
	_commonInfo CommonInfo
	_commonInfoStr string

	_reportCounter uint16
	_reportMutex *sync.Mutex
	_startTime time.Time
	_reportCounterLimit uint16 // 10分钟之内最大上报数，超过的直接丢弃

	_initCommonInfoStrOnce sync.Once
)

func init() {
	_reportCounter = 0
	_reportCounterLimit = 100
	_reportMutex = &sync.Mutex{}
}


