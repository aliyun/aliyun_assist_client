package acspluginmanager

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
)

func getArch() (formatArch string, rawArch string) {
	defer func() {
		log.GetLogger().Errorf("Get Arch: formatArch[%s] rawArch[%s]: ", formatArch, rawArch)
	}()
	formatArch = ARCH_UNKNOWN
	arch, err := osutil.GetUnameMachine()
	if err != nil {
		log.GetLogger().Errorln("Get Arch: GetUnameMachine err: ", err.Error())
	}
	arch = strings.TrimSpace(arch)
	arch = strings.ToLower(arch)
	rawArch = arch

	if strings.Contains(arch, "aarch") || strings.Contains(arch, "arm"){ // arm: aarch arm
		formatArch = ARCH_ARM
	} else if strings.Contains(arch, "386") || strings.Contains(arch, "686") { // x86: i386 i686
		formatArch = ARCH_32
	} else if  arch == "x86_64" { // x64: x86_64
		formatArch = ARCH_64
	} else {
		log.GetLogger().Errorln("Get Arch: unknown arch: ", arch)
		formatArch = ARCH_UNKNOWN
	}
	return
}