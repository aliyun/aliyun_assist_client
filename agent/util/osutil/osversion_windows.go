// +build windows

package osutil

import (
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/shirou/gopsutil/host"
)

func getVersion() string {
	// The windows version just uses PlatformInformation() from shirou/gopsutil
	// library, which may produce different result from C++ agent. To consturct
	// version string manually, see implementation detail of PlatformInformation()
	// patiently.
	platform, _, _, err := host.PlatformInformation()
	if err != nil {
		return "unknown OperatingSystem."
	}
	return platform
}

func getKernelVersion() string {
	kernelVersion, err := host.KernelVersion()
	if err != nil {
		log.GetLogger().Error("get kernel version error: ", err)
		return "unknown KernelVersion"
	}
	kernelVersion = strings.TrimSpace(kernelVersion)
	return kernelVersion
}