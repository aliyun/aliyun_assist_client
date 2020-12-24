package clientreport

import (
	"encoding/json"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/sirupsen/logrus"
)

type panicInfo struct {
	Panic      string `json:"panic"`
	Stacktrace string `json:"stacktrace"`
}

func LogAndReportPanic(payload interface{}, stacktrace []byte) {
	info := panicInfo{
		Panic:      fmt.Sprint(payload),
		Stacktrace: string(stacktrace),
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

	log.GetLogger().Fatalf("panic: %v\n\n%s", payload, stacktrace)
}
