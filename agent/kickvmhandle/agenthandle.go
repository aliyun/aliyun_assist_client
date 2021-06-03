package kickvmhandle

import (
	"errors"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"os"
	"runtime"
	"strings"
)
// type:agent
var agentRoute map[string]handleFunc
func init ()  {
	agentRoute = map[string]handleFunc{
		"stop": stopAgant,
		"remove": removeAgant,
		"update": updateAgant,
	}
}

func stopAgant(params []string) error {
	log.GetLogger().Println("stopAgant")
	processer :=  process.ProcessCmd{}
	if runtime.GOOS == "linux" {
		processer.SyncRunSimple("aliyun-service", strings.Split("--stop", " "), 10)
	} else if runtime.GOOS == "windows" {
		path, err := os.Executable()
		if err != nil {
			return err
		}
		processer.SyncRunSimple(path, strings.Split("--stop", " "), 10)
	}
	return nil
}

func removeAgant(params []string) error {
	log.GetLogger().Println("removeAgant")
	processer :=  process.ProcessCmd{}
	if runtime.GOOS == "linux" {
		processer.SyncRunSimple("aliyun-service", strings.Split("--remove", " "), 10)
		processer.SyncRunSimple("aliyun-service", strings.Split("--stop", " "), 10)
	} else if runtime.GOOS == "windows" {
		path, err := os.Executable()
		if err != nil {
			return err
		}
		processer.SyncRunSimple(path, strings.Split("--remove", " "), 10)
		processer.SyncRunSimple(path, strings.Split("--stop", " "), 10)
	}
	return nil
}

func updateAgant(params []string) error {
	log.GetLogger().Println("updateAgant")
	processer :=  process.ProcessCmd{}
	path, err := util.GetCurrentPath()
	if err != nil {
		return err
	}
	if runtime.GOOS == "linux" {
		path += "aliyun_assist_update"
	} else {
		path += "aliyun_assist_update.exe"
	}

	processer.SyncRunSimple(path, strings.Split("--check_update", " "), 10)
	return nil
}

type AgentHandle struct {
	action string
	params []string
}

func NewAgentHandle(action string, params []string) *AgentHandle{
	return  &AgentHandle{
		action: action,
		params: params,
	}
}


func (h *AgentHandle) DoAction() error{
	if v, ok := agentRoute[h.action]; ok {
		v(h.params)
	} else {
		return errors.New("no action found")
	}
	return nil
}

func (h *AgentHandle) CheckAction() bool{
	if _, ok := agentRoute[h.action]; ok {
		return true
	} else {
		return false
	}
}