package powerutil

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

const (
	PowerdownMode = "powerdown"
	RebootMode = "reboot"
)

func Shutdown(reboot bool) {
	command := powerdownCmd
	mode := PowerdownMode
	if reboot {
		command = rebootCmd
		mode = RebootMode
	}
	log.GetLogger().Infof("%s......", mode)
	go func() {
		time.Sleep(100 * time.Millisecond)
		err, stdout, stderr := util.ExeCmd(command)
		if err != nil {
			metrics.GetShutDownFailedEvent(
				"mode", mode,
				"error", err.Error(),
				"stdout", stdout,
				"stderr", stderr,
			).ReportEvent()
		}
	}()
}