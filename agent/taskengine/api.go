package taskengine

import (
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func sendStoppedOutput(taskId string, start int64, end int64, exitcode int,
	dropped int, output string) (string, error) {
	path := util.GetStoppedOutputService()
	// luban/api/v1/task/stopped API requires extra result=killed parameter in
	// querystring
	querystring := fmt.Sprintf("?taskId=%s&start=%d&end=%d&exitcode=%d&dropped=%d&result=killed",
		taskId, start, end, exitcode, dropped)
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
