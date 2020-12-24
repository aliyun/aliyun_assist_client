package taskengine

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/tidwall/gjson"
)

type taskInfo struct {
	TaskInfo   RunTaskInfo `json:"task"`
	OutputInfo OutputInfo  `json:"output"`
}

type sendFileInfo struct {
	TaskInfo   SendFileTaskInfo `json:"task"`
	OutputInfo OutputInfo       `json:"output"`
}

type tasks struct {
	RunTasks      []taskInfo     `json:"run"`
	StopTasks     []taskInfo     `json:"stop"`
	SendFileTasks []sendFileInfo `json:"file"`
	InstanceId    string         `json:"instanceId"`
}

func parseTaskInfo(jsonStr string) ([]RunTaskInfo, []RunTaskInfo, []SendFileTaskInfo) {
	var runInfos []RunTaskInfo
	var stopInfos []RunTaskInfo
	var sendFiles []SendFileTaskInfo

	if !gjson.Valid(jsonStr) {
		fmt.Println("invalid task info json:", jsonStr)
		return runInfos, stopInfos, sendFiles
	}

	var task_lists tasks
	if err := json.Unmarshal([]byte(jsonStr), &task_lists); err == nil {
		for _, v := range task_lists.RunTasks {
			v.TaskInfo.Output = v.OutputInfo
			runInfos = append(runInfos, v.TaskInfo)
		}

		for _, stopTask := range task_lists.StopTasks {
			stopTaskInfo := stopTask.TaskInfo
			stopTaskInfo.Output = stopTask.OutputInfo
			stopInfos = append(stopInfos, stopTaskInfo)
		}
		for _, sendFileTask := range task_lists.SendFileTasks {
			sendFile := sendFileTask.TaskInfo
			sendFile.Output = sendFileTask.OutputInfo
			sendFiles = append(sendFiles, sendFile)
		}
	}

	return runInfos, stopInfos, sendFiles
}

func FetchTaskList(reason string) ([]RunTaskInfo, []RunTaskInfo, []SendFileTaskInfo) {
	var runInfos []RunTaskInfo
	var stopInfos []RunTaskInfo
	var sendFiles []SendFileTaskInfo

	if util.GetServerHost() == "" {
		return runInfos, stopInfos, sendFiles
	}

	url := util.GetFetchTaskListService()
	url = url + "?reason=" + reason

	var err error
	var response string
	response, err = util.HttpPost(url, "", "")

	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		response, err = util.HttpPost(url, "", "")
	}

	if err != nil {
		return runInfos, stopInfos, sendFiles
	}

	runInfos, stopInfos, sendFiles = parseTaskInfo(response)
	return runInfos, stopInfos, sendFiles
}
