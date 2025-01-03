package daemon

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/systemdutil"
	"github.com/aliyun/aliyun_assist_client/common/executil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

// Daemonize runs this program as daemon.
// Traditional unix's fork-style way would be dangerous to damonize Go program
// since it may break states of underlying runtime scheduler of goroutines. Thus
// simple initiate another process of this program with special setting.
func Daemonize() error {
	executablePath := os.Args[0]
	cmd := executil.Command(executablePath)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	err := cmd.Start()
	if err != nil {
		return err
	}

	cmd.Process.Release()
	os.Exit(0)
	return nil
}

const (
	AssistDaemonUnit = "AssistDaemon.service"
)

func OperateAssistDaemon(logger logrus.FieldLogger, v any) {
	if !systemdutil.IsRunningSystemd() {
		return
	}
	var active, ok bool
	if active, ok = v.(bool); !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if active {
		if res, err := systemdutil.StartUnit(ctx, AssistDaemonUnit); err != nil {
			logger.WithError(err).Errorf("Start %s unit failed", AssistDaemonUnit)
		} else {
			logger.Infof("Start %s unit: %s", AssistDaemonUnit, res)
		}
	} else {
		if res, err := systemdutil.StopUnit(ctx, AssistDaemonUnit); err != nil {
			logger.WithError(err).Errorf("Stop %s unit failed", AssistDaemonUnit)
		} else {
			logger.Infof("Stop %s unit: %s", AssistDaemonUnit, res)
		}
	}
}
