package kickvmhandle

import (
	"errors"
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