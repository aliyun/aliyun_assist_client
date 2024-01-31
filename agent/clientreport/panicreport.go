package clientreport

import (
	"encoding/json"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type panicInfo struct {
	Panic      string `json:"panic"`
	Stacktrace string `json:"stacktrace"`
	Ignore     bool   `json:"ignore"`
}

// LogAndReportPanic reports panic to server and log to file then exit
func LogAndReportPanic(payload interface{}, stacktrace []byte) {
	ReportPanic(payload, stacktrace, true)
}

func LogAndReportIgnorePanic(payload interface{}, stacktrace []byte) {
	ReportPanic(payload, stacktrace, false)
}

// ReportPanic reports panic to server and log to file, exit program or ignore according to exit parameter
func ReportPanic(payload interface{}, stacktrace []byte, exit bool) {
	info := panicInfo{
		Panic:      fmt.Sprint(payload),
		Stacktrace: string(stacktrace),
		Ignore:     !exit,
	}
	infoJSONBytes, err := json.Marshal(info)
	if err != nil {
		log.GetLogger().WithError(err).Errorln("Failed to stringify panic information")
	} else {
		report := ClientReport{
			ReportType: "AgentPanic",
			Info:       string(infoJSONBytes),
		}
		_, err = SendReport(report)
		if err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"report": report,
			}).WithError(err).Errorln("Failed to send panic report")
		}
	}
	if exit {
		log.GetLogger().Fatalf("panic: %v\n\n%s", payload, stacktrace)
	} else {
		log.GetLogger().Errorf("panic ignored: %v\n\n%s", payload, stacktrace)
	}
}
