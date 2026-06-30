//go:build linux || darwin || freebsd
// +build linux darwin freebsd

package pluginmodel

import (
	"strings"
)

func HostArchitecture2Model(arch string) (string, bool) {
	if strings.Contains(arch, "aarch") || strings.Contains(arch, "arm") { // arm: aarch arm
		return ARCH_ARM, true
	} else if strings.Contains(arch, "386") || strings.Contains(arch, "686") { // x86: i386 i686
		return ARCH_32, true
	} else if arch == "x86_64" || arch == "amd64" { // x64: x86_64
		return ARCH_64, true
	} else {
		return ARCH_UNKNOWN, false
	}
}
