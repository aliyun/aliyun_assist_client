//go:build windows
// +build windows

package serviceutil

import (
	"strconv"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// RestartService restarts Agent service, the Agent service process itself
// should not call this function
func RestartAgentService(logger logrus.FieldLogger) {
	s, err := getService()
	if err != nil {
		logger.WithError(err).Error("failed to get service handle")
		goto RESTARTBYCOMMAND
	}
	if err = StopWait(s); err != nil {
		logger.WithError(err).Error("failed to stop service by service handle")
		goto RESTARTBYCOMMAND
	}
	if err = s.Start(); err != nil {
		logger.WithError(err).Error("failed to start service by service handle")
		goto RESTARTBYCOMMAND
	}
	return
RESTARTBYCOMMAND:
	logger.Info("try to restart service by command")
	if err := restartServiceByCommand(); err != nil {
		logger.WithError(err).Error("failed to restart service by command")
	}
}

// StopAgentService stops Agent service
func StopAgentService(logger logrus.FieldLogger) {
	s, err := getService()
	if err != nil {
		logger.WithError(err).Error("failed to get service handle")
		goto STOPBYCOMMAND
	} 
	if err = StopWait(s); err != nil {
		logger.WithError(err).Error("failed to stop service by service handle")
		goto STOPBYCOMMAND
	}
	return
STOPBYCOMMAND:
	logger.Info("try to stop service by command")
	if err := stopServiceByCommand(); err != nil {
		logger.WithError(err).Error("failed to stop service by command")
	}
}

func stopServiceByCommand() error {
	processer := process.ProcessCmd{}
	return processer.SyncRunSimple("net", []string{"stop", "AliyunService"}, 10)
}

func restartServiceByCommand() error {
	processer := process.ProcessCmd{}
	processer.SyncRunSimple("net", []string{"stop", "AliyunService"}, 10)
	return processer.SyncRunSimple("net", []string{"start", "AliyunService"}, 10)
}

func getService() (*mgr.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()
	return m.OpenService("AliyunService")
}

func StopWait(s *mgr.Service) error {
	// First stop the service. Then wait for the service to
	// actually stop before starting it.
	status, err := s.Control(svc.Stop)
	if err != nil {
		return err
	}

	timeDuration := time.Millisecond * 50

	timeout := time.After(getStopTimeout() + (timeDuration * 2))
	tick := time.NewTicker(timeDuration)
	defer tick.Stop()

	for status.State != svc.Stopped {
		select {
		case <-tick.C:
			status, err = s.Query()
			if err != nil {
				return err
			}
		case <-timeout:
			break
		}
	}
	return nil
}

// getStopTimeout fetches the time before windows will kill the service.
func getStopTimeout() time.Duration {
	// For default and paths see https://support.microsoft.com/en-us/kb/146092
	defaultTimeout := time.Millisecond * 20000
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control`, registry.READ)
	if err != nil {
		return defaultTimeout
	}
	defer key.Close()
	sv, _, err := key.GetStringValue("WaitToKillServiceTimeout")
	if err != nil {
		return defaultTimeout
	}
	v, err := strconv.Atoi(sv)
	if err != nil {
		return defaultTimeout
	}
	return time.Millisecond * time.Duration(v)
}
