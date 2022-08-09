package taskengine

import (
	"fmt"
	"net/url"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

const (
	invalidParamCron string = "cron"

	stopReasonKilled string = "killed"
	stopReasonCompleted string = "completed"
)

func reportInvalidTask(taskId string, param string, value string) (string, error) {
	escapedParam := url.QueryEscape(param)
	escapedValue := url.QueryEscape(value)
	path := util.GetInvalidTaskService()
	querystring := fmt.Sprintf("?taskId=%s&param=%s&value=%s", taskId, escapedParam, escapedValue)
	url := path + querystring

	var response string
	var err error
	response, err = util.HttpPost(url, "", "text")
	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		response, err = util.HttpPost(url, "", "text")
	}

	return response, err
}

func sendStoppedOutput(taskId string, start int64, end int64, exitcode int,
	dropped int, output string, reason string) (string, error) {
	path := util.GetStoppedOutputService()
	// luban/api/v1/task/stopped API requires extra result=killed parameter in
	// querystring
	querystring := fmt.Sprintf("?taskId=%s&start=%d&end=%d&exitcode=%d&dropped=%d&result=%s",
		taskId, start, end, exitcode, dropped, reason)
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
