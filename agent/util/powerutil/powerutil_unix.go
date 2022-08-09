//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package powerutil

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func Powerdown() {
	shutdownCmd := "shutdown -h now"
	log.GetLogger().Infoln("powerdown......")
	util.ExeCmd(shutdownCmd)
}

func Reboot() {
	rebootCmd := "shutdown -r now"
	log.GetLogger().Infoln("reboot......")
	util.ExeCmd(rebootCmd)
}
