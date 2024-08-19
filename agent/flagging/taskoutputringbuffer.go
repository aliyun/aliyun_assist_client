package flagging

import (
	"go.uber.org/atomic"
)

const (
	disableTaskOutputRingbufferFlagFilename = "disable_taskoutput_ringbuffer"
	disableTaskOutputRingbufferFlagName     = "Disable TaskOutput-Ringbuffer"
)

var (
	_taskOutputRingbufferDisabled atomic.Bool
)

func IsTaskOutputRingbufferDisabled() bool {
	return _taskOutputRingbufferDisabled.Load()
}

func DetectTaskOutputRingbufferDisabled() (bool, error) {
	return detectFlagSet(disableTaskOutputRingbufferFlagName, disableTaskOutputRingbufferFlagFilename, &_taskOutputRingbufferDisabled)
}
