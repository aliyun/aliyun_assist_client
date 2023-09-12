package taskengine

import (
	"encoding/json"
	"fmt"
	neturl "net/url"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
)

type FetchReason string

const (
	FetchOnKickoff FetchReason = "kickoff"
	FetchOnStartup FetchReason = "startup"
)

type taskInfo struct {
	TaskInfo   models.RunTaskInfo       `json:"task"`
	OutputInfo models.OutputInfo        `json:"output"`
	Repeat     models.RunTaskRepeatType `json:"repeat"`
}

type sendFileInfo struct {
	TaskInfo   models.SendFileTaskInfo `json:"task"`
	OutputInfo models.OutputInfo       `json:"output"`
}

type tasks struct {
	Code          int                      `json:"code"`
	RunTasks      []taskInfo               `json:"run"`
	StopTasks     []taskInfo               `json:"stop"`
	TestTasks     []taskInfo               `json:"test"`
	SendFileTasks []sendFileInfo           `json:"file"`
	SessionTasks  []models.SessionTaskInfo `json:"session"`
	InstanceId    string                   `json:"instanceId"`
}

type taskCollection struct {
	runInfos     []models.RunTaskInfo
	stopInfos    []models.RunTaskInfo
	testInfos    []models.RunTaskInfo
	sendFiles    []models.SendFileTaskInfo
	sessionInfos []models.SessionTaskInfo
}

func newTaskCollection() *taskCollection {
	taskInfos := taskCollection{
		runInfos:     []models.RunTaskInfo{},
		stopInfos:    []models.RunTaskInfo{},
		testInfos:    []models.RunTaskInfo{},
		sendFiles:    []models.SendFileTaskInfo{},
		sessionInfos: []models.SessionTaskInfo{},
	}
	return &taskInfos
}

func parseTaskInfo(jsonStr string) (int, *taskCollection) {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"module": "parseTaskInfo",
	})

	taskInfos := newTaskCollection()

	var task_lists tasks
	err := json.Unmarshal([]byte(jsonStr), &task_lists)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"jsonString": jsonStr,
		}).WithError(err).Errorln("Invalid task info json")
		return 0, taskInfos
	}

	for _, v := range task_lists.RunTasks {
		runTaskInfo, err := v.toRunTaskInfo(task_lists.InstanceId)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"runTask": v,
			}).WithError(err).Errorln("Invalid run task info")
			continue
		}
		taskInfos.runInfos = append(taskInfos.runInfos, runTaskInfo)
	}

	for _, stopTask := range task_lists.StopTasks {
		stopTaskInfo, err := stopTask.toRunTaskInfo(task_lists.InstanceId)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"stopTask": stopTask,
			}).WithError(err).Errorln("Invalid stop task info")
			continue
		}
		taskInfos.stopInfos = append(taskInfos.stopInfos, stopTaskInfo)
	}
	for _, testTask := range task_lists.TestTasks {
		testTaskInfo, err := testTask.toRunTaskInfo(task_lists.InstanceId)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"testTask": testTask,
			}).WithError(err).Errorln("Invalid test task info")
			continue
		}
		taskInfos.testInfos = append(taskInfos.testInfos, testTaskInfo)
	}
	for _, sendFileTask := range task_lists.SendFileTasks {
		sendFile := sendFileTask.TaskInfo
		sendFile.Output = sendFileTask.OutputInfo
		taskInfos.sendFiles = append(taskInfos.sendFiles, sendFile)
	}

	for _, sessionTask := range task_lists.SessionTasks {
		taskInfos.sessionInfos = append(taskInfos.sessionInfos, sessionTask)
	}

	return task_lists.Code, taskInfos
}

func FetchTaskList(reason FetchReason, taskId string, taskType int, isColdstart bool) *taskCollection {
	if util.GetServerHost() == "" {
		return newTaskCollection()
	}

	url := util.GetFetchTaskListService()
	switch reason {
	case FetchOnKickoff:
		url = url + "?reason=" + string(reason)
	case FetchOnStartup:
		url = url + fmt.Sprintf("?reason=%s&cold_start=%t", reason, isColdstart)
	default:
		log.GetLogger().WithFields(logrus.Fields{
			"reason": reason,
		}).Errorln("Invalid reason for fetching tasks")
		return newTaskCollection()
	}
	if taskType == SessionTaskType {
		url = util.GetFetchSessionTaskListService()
		if taskId != "" {
			url = url + "?channelId=" + taskId
		}
	} else {
		if taskId != "" {
			url = url + "&taskId=" + taskId
		}
		// Append Unix timestamp and timezone name of current wall clock
		currentTime, currentOffsetFromUTC, timezoneName := timetool.NowWithTimezoneName()
		escapedTimezoneName := neturl.QueryEscape(timezoneName)
		url += fmt.Sprintf("&currentTime=%d&offset=%d&timeZone=%s", timetool.ToAccurateTime(currentTime), currentOffsetFromUTC, escapedTimezoneName)
	}

	var err error
	var response string
	var taskInfos *taskCollection
	var code int
	for idx := 0; idx < 4; idx++ {
		response, err = util.HttpPostWithTimeout(url, "", "", 8, false)
		for i := 0; i < 3 && err != nil; i++ {
			time.Sleep(time.Duration(2) * time.Second)
			response, err = util.HttpPostWithTimeout(url, "", "", 8, false)
		}
		if err != nil {
			return newTaskCollection()
		}
		code, taskInfos = parseTaskInfo(response)
		if code == 408 {
			time.Sleep(time.Duration(2) * time.Second)
			continue
		}
		break
	}

	return taskInfos
}

func (t *taskInfo) toRunTaskInfo(instanceId string) (models.RunTaskInfo, error) {
	runTaskInfo := t.TaskInfo
	runTaskInfo.InstanceId = instanceId
	runTaskInfo.Output = t.OutputInfo
	runTaskInfo.Repeat = t.Repeat

	// Compatible with no `Repeat` field in task info pulled
	// TODO: Remove compatibility code when `Repeat` field fully available
	if runTaskInfo.Repeat == "" {
		if runTaskInfo.Cronat != "" {
			runTaskInfo.Repeat = models.RunTaskCron
		} else {
			runTaskInfo.Repeat = models.RunTaskOnce
		}
	}

	// Prepare values of environment parameters if enableParameter is true
	if runTaskInfo.EnableParameter {
		if runTaskInfo.BuiltinParameters == nil {
			runTaskInfo.BuiltinParameters = make(map[string]string, 3)
		}
		runTaskInfo.BuiltinParameters["InstanceId"] = instanceId
		runTaskInfo.BuiltinParameters["CommandId"] = runTaskInfo.CommandId
		runTaskInfo.BuiltinParameters["InvokeId"] = runTaskInfo.TaskId
	}

	return runTaskInfo, nil
}
