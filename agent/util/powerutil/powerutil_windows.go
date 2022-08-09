// +build windows

package powerutil

import (
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func Powerdown() {
	shutdownCmd := "shutdown -f -s -t 0"
	log.GetLogger().Infoln("powerdown......")
	util.ExeCmd(shutdownCmd)
}

func Reboot() {
	rebootCmd := "shutdown -f -r -t 0"
	log.GetLogger().Infoln("reboot......")
	util.ExeCmd(rebootCmd)
}
