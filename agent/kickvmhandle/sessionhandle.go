package kickvmhandle

import (
	"errors"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine"
)

var sessionRoute map[string]handleFunc
func init ()  {
	sessionRoute = map[string]handleFunc{
		"start": startSession,
		"stop": stopSession,
	}
}

func stopSession(params []string) error {
	go func() {
		if len(params) < 1 {
			log.GetLogger().Errorln("params invalid", params)
			return
		}
		ret := taskengine.GetSessionFactory().ContainsTask(params[0])
		if ret == true {
			log.GetLogger().Println("stop session ", params[0])
			task,_ := taskengine.GetSessionFactory().GetTask(params[0])
			task.StopTask()
		} else {
			log.GetLogger().Errorln("stop session failed")
		}
	}()
	return nil
}

func startSession(params []string) error {
	go func() {
		// kick_vm session  start task_id
		// kick_vm session  stop task_id
		if len(params) < 1 {
			log.GetLogger().Errorln("params invalid", params)
			return
		}
		taskengine.Fetch(true, params[0], taskengine.SessionTaskType, false)
	}()
	return nil
}

type SessionHandle struct {
	action string
	params []string
}

func NewSessionHandle(action string, params []string) *SessionHandle{
	return  &SessionHandle{
		action: action,
		params: params,
	}
}

func (h *SessionHandle) DoAction() error{
	if v, ok := sessionRoute[h.action]; ok {
		v(h.params)
	} else {
		return errors.New("no action found")
	}
	return nil
}

func (h *SessionHandle) CheckAction() bool{
	if _, ok := sessionRoute[h.action]; ok {
		return true
	} else {
		return false
	}
}