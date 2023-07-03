package clientreport

import (
	"encoding/json"
	"math"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type ClientReport struct {
	ReportType string `json:"type"`
	Info       string `json:"info"`
}

// SendReport marshals client report data and sends to server
func SendReport(report ClientReport) (string, error) {
	requestPayloadBytes, err := json.Marshal(report)
	if err != nil {
		return "", err
	}
	requestPayload := string(requestPayloadBytes)

	requestURL := util.GetClientReportService()
	response, err := util.HttpPost(requestURL, requestPayload, "")
	for i := 0; i < 3 && err != nil; i++ {
		sleepDuration := time.Duration(math.Pow(2, float64(i))) * time.Second
		time.Sleep(sleepDuration)

		response, err = util.HttpPost(requestURL, requestPayload, "")
	}
	if err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"requestURL":     requestURL,
			"requestPayload": requestPayload,
		}).Errorln("Network is unavailable for client report request")
		return "", err
	}

	return response, nil
}
