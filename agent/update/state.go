package update

import (
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
)

var (
	_cpuIntensiveActionRunning atomicutil.AtomicBoolean
	_criticalActionRunning atomicutil.AtomicBoolean
)

// IsCPUIntensiveActionRunning returns cpuIntensiveActionRunning flag as boolean variable
func IsCPUIntensiveActionRunning() bool {
	return _cpuIntensiveActionRunning.IsSet()
}

// IsCriticalActionRunning returns criticalActionRunning flag as boolean variable
func IsCriticalActionRunning() bool {
	return _criticalActionRunning.IsSet()
}
