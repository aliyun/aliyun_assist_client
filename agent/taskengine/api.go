package taskengine

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
)

const (
	invalidParamCron string = "cron"

	stopReasonKilled    string = "killed"
	stopReasonCompleted string = "completed"
	stopFailed          string = "failed"
)

const (
	TaskReportErrTaskNotFound      = "TaskNotFound"
	TaskReportErrTaskStatusInvalid = "TaskStatusInvalid"
	TaskInvalidStatusTerminated    = "Terminated"
)

type TaskReportResp struct {
	ErrorCode string `json:"errorCode"`
	Status    string `json:"status"`
}

func reportInvalidTask(taskId string, invokeVersion int, param, value string, extraLubanParams string) (string, error) {
	escapedParam := url.QueryEscape(param)
	escapedValue := url.QueryEscape(value)
	path := util.GetInvalidTaskService()
	querystring := fmt.Sprintf("?taskId=%s&invokeVersion=%d&param=%s&value=%s",
		taskId, invokeVersion, escapedParam, escapedValue)
	url := path + querystring
	url += extraLubanParams

	var response string
	var err error
	response, err = util.HttpPost(url, "", "text")
	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		response, err = util.HttpPost(url, "", "text")
	}

	return response, err
}

func sendStoppedOutput(taskId string, invokeVersion int, start int64, end int64, exitcode int,
	dropped int, output string, reason string, stopErr error) (string, error) {
	path := util.GetStoppedOutputService()
	// luban/api/v1/task/stopped API requires extra result=killed parameter in
	// querystring
	querystring := fmt.Sprintf("?taskId=%s&invokeVersion=%d&start=%d&end=%d&exitcode=%d&dropped=%d&result=%s",
		taskId, invokeVersion, start, end, exitcode, dropped, reason)
	if stopErr != nil {
		errDesc := stopErr.Error()
		safelyTruncatedErrDesc := langutil.SafeTruncateStringInBytes(errDesc, 255)
		escapedErrDesc := url.QueryEscape(safelyTruncatedErrDesc)
		querystring = querystring + "&errCode=TerminationException&errDesc=" + escapedErrDesc
	}
	url := path + querystring

	var response string
	var err error
	response, err = util.HttpPost(url, output, "text")
	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		response, err = util.HttpPost(url, output, "text")
	}

	return response, err
}

func parseTaskReportResp(content string) *TaskReportResp {
	resp := &TaskReportResp{}
	if err := json.Unmarshal([]byte(content), resp); err != nil {
		log.GetLogger().WithError(err).Error("Marshal TaskReportResp failed")
	}
	return resp
}
