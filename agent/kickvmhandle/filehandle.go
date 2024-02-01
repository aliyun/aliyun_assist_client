package kickvmhandle

import (
	"errors"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
)

// type:agent

var fileRoute map[string]handleFunc
func init ()  {
	fileRoute = map[string]handleFunc{
		"create": runFileTask,
		"stop": stopFileTask,
	}
}

func runFileTask(params []string) error {
	log.GetLogger().Println("runFileTask")
	if len(params) < 1 {
		return errors.New("params error")
	}
	go func() {
		taskengine.Fetch(true, params[0], taskengine.NormalTaskType)
	}()
	return nil
}

func stopFileTask(params []string) error {
	log.GetLogger().Println("stopFileTask")
	if len(params) < 1 {
		return errors.New("params error")
	}

	go func() {
		taskengine.Fetch(true, params[0], taskengine.NormalTaskType)
	}()
	return nil
}


type FileHandle struct {
	action string
	params []string
}

func NewFileHandle(action string, params []string) *FileHandle{
	return  &FileHandle{
		action: action,
		params: params,
	}
}

func (h *FileHandle) DoAction() error{
	if v, ok := fileRoute[h.action]; ok {
		v(h.params)
	} else {
		return errors.New("no action found")
	}
	return nil
}

func (h *FileHandle) CheckAction() bool{
	if _, ok := fileRoute[h.action]; ok {
		return true
	} else {
		return false
	}
}