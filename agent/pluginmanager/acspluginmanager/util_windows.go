package acspluginmanager

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

func getArch() (formatArch string, rawArch string) {
	// 云助手的windows版架构只有amd64的
	formatArch = ARCH_64
	rawArch = "windows arch"
	log.GetLogger().Errorf("Get Arch: formatArch[%s] rawArch[%s]: ", formatArch, rawArch)
	return
}