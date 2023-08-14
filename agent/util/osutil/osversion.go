package osutil

import (
	"sync"
)

var (
	_initVersionOnce sync.Once
	_initKernelVersionOnce sync.Once

	_version string
	_kernelVersion string
)

func GetVersion() string {
	_initVersionOnce.Do(func() {
		_version = getVersion()
	})
	return _version
}

func GetKernelVersion() string {
	_initKernelVersionOnce.Do(func() {
		_kernelVersion = getKernelVersion()
	})
	return _kernelVersion
}