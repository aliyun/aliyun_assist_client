package export

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type ExporterResult struct {
	TaskId       string          `json:"taskId"`
	OssExportRes []*OSSExportRes `json:"ossExporter"`
}

func ExportToOss(logger logrus.FieldLogger, taskId string, invokeVersion int, content []byte, filePath string, fileCreateErr error, fileWriteErr error, confList []*OSSExporter) {
	res := &ExporterResult{
		TaskId: taskId,
	}
	var r *OSSExportRes
	for _, exporter := range confList {
		if content != nil {
			r = exporter.PutObjectFromContent(logger, content)
		} else {
			if fileCreateErr != nil {
				r = exporter.FailedCreateOutputFile(fileCreateErr)
			} else {
				r = exporter.PutObjectFromFile(logger, filePath)
				if r.StatusCode == 200 && fileWriteErr != nil {
					r = exporter.FailedWriteOutputFile(fileWriteErr)
				}
			}
		}
		res.OssExportRes = append(res.OssExportRes, r)
	}

	url := util.GetExportOutputResultService()
	url += fmt.Sprintf("?taskId=%s&invokeVersion=%d",
		taskId, invokeVersion)
	content, err := json.Marshal(res)
	if err != nil {
		logger.WithError(err).Error("Marshal ExporterResult failed")
		return
	}
	_, err = util.HttpPost(url, string(content), "json")
	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		_, err = util.HttpPost(url, string(content), "json")
	}
	if err != nil {
		logger.WithError(err).Error("Report exporter result failed")
	}
}
