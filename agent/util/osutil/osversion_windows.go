package osutil

import (
	"github.com/shirou/gopsutil/host"
)

func GetVersion() string {
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
