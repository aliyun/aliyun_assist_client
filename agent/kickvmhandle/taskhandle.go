package kickvmhandle

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
)

// type:agent

var taskRoute map[string]handleFunc
func init ()  {
	taskRoute = map[string]handleFunc{
		"run": runTask,
		"stop": stopTask,
	}
}

func runTask(params []string) error {
	log.GetLogger().Println("runTask")
	if len(params) < 1 {
		return errors.New("params error")
	}
	go func() {
		taskengine.Fetch(true, params[0], taskengine.NormalTaskType)	}()
	return nil
}

func stopTask(params []string) error {
	log.GetLogger().Println("stopTask")
	if len(params) < 1 {
		return errors.New("params error")
	}

	go func() {
		taskengine.Fetch(true, params[0], taskengine.NormalTaskType)	}()
	return nil
}


type TaskHandle struct {
	action string
	params []string
}

func NewTaskHandle(action string, params []string) *TaskHandle{
	return  &TaskHandle{
		action: action,
		params: params,
	}
}

func (h *TaskHandle) DoAction() error{
	if v, ok := taskRoute[h.action]; ok {
		v(h.params)
	} else {
		return errors.New("no action found")
	}
	return nil
}

func (h *TaskHandle) CheckAction() bool{
	if _, ok := taskRoute[h.action]; ok {
		return true
	} else {
		return false
	}
}