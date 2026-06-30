package checknet

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/serialport"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	_refreshNetcheckTimeThreshold = time.Duration(15) * time.Minute

	_reportToSerialPortTimeThreshold = time.Hour
	_reportNetworkBlockPrefix        = "AliyunAssist: NetworkBlock"
	_reportNoNetworkCollectResPrefix = "AliyunAssist: NoNetworkCollectRes"
)

var (
	_lastTimeReportNetworkBlockToSerialPort              *time.Time
	_lastReportNetworkBlockToSerialPortInProgress        atomic.Bool
	_lastReportNoNetworkCollectResToSerialPortInProgress atomic.Bool
)

type CheckReport struct {
	Result       int
	FinishedTime time.Time
}

func isReportOutdated(reportedTime time.Time) bool {
	return time.Now().Local().Sub(reportedTime.Local()) >= _refreshNetcheckTimeThreshold
}

// ReportNetworkBlockToSerialPort print heart beat error to serialport
func ReportNetworkBlockToSerialPort(heartbeatErr error) error {
	// Do not write anything into serialport in hybrid instance.
	if instance.IsHybrid() {
		return nil
	}

	logger := log.GetLogger().WithField("reportToSerialPort", "NetworkBlock")
	if !_lastReportNetworkBlockToSerialPortInProgress.CompareAndSwap(false, true) {
		logger.Warn("Another reporting is in process, skip.")
		return nil
	}
	defer _lastReportNetworkBlockToSerialPortInProgress.Store(false)

	if _lastTimeReportNetworkBlockToSerialPort == nil || time.Since(*_lastTimeReportNetworkBlockToSerialPort) >= _reportToSerialPortTimeThreshold {
		t := time.Now()
		_lastTimeReportNetworkBlockToSerialPort = &t
		return reportToSerialPort(logger, _reportNetworkBlockPrefix, heartbeatErr.Error())
	}
	return nil
}

func ReportNoNetworkCollectResToSerialPort(content string) {
	// Do not write anything into serialport in hybrid instance.
	if instance.IsHybrid() {
		return
	}

	logger := log.GetLogger().WithField("reportToSerialPort", "NoNetworkCollectRes")
	if !_lastReportNoNetworkCollectResToSerialPortInProgress.CompareAndSwap(false, true) {
		logger.Warn("Another reporting is in process, skip.")
		return
	}
	defer _lastReportNoNetworkCollectResToSerialPortInProgress.Store(false)

	reportToSerialPort(logger, _reportNoNetworkCollectResPrefix, content)
}

func reportToSerialPort(logger logrus.FieldLogger, prefix string, content string) error {
	sp, err := serialport.GetSerialPort()
	if err != nil {
		logger.WithError(err).Error("Report to serial port failed.")
		return err
	}
	defer sp.ClosePort()
	// In console_log, a timestamp may be inserted before the content.
	// Add a space here to prevent word segmentation from failing.
	allContent := fmt.Sprintf(" %s: %s", prefix, content)
	logger.Info("Report to serial port succeeded.")
	return sp.WritePort([]byte(allContent))
}
