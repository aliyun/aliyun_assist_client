package checknet

import (
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/serialport"
)

const (
	_refreshNetcheckTimeThreshold = time.Duration(15) * time.Minute

	_reportToSerialPortTimeThreshold = time.Hour
	_reportNetworkBlockPrefix        = "AliyunAssist: NetworkBlock"
)

var (
	_lastTimeReportNetworkBlockToSerialPort *time.Time
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
	if _lastTimeReportNetworkBlockToSerialPort == nil || time.Since(*_lastTimeReportNetworkBlockToSerialPort) >= _reportToSerialPortTimeThreshold {
		t := time.Now()
		_lastTimeReportNetworkBlockToSerialPort = &t
		sp, err := serialport.GetSerialPort()
		if err != nil {
			return err
		}
		defer sp.ClosePort()
		// In console_log, a timestamp may be inserted before the content.
		// Add a space here to prevent word segmentation from failing.
		content := fmt.Sprintf(" %s: %v", _reportNetworkBlockPrefix, heartbeatErr)
		log.GetLogger().Info("Report network block to serial port.")
		return sp.WritePort([]byte(content))
	}
	return nil
}
