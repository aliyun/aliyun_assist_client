package kickvmhandle

import (
	"strings"
)

// kick_vm kick_type action params...
// for example: kick_vm  task  run  t-xxxxxxxx
type handleFunc func(params []string) error

type KickHandle interface {
	DoAction() error
	CheckAction() bool
}

func (h *HealthCheckHandle) DoAction() error{

	return nil
}

func (h *HealthCheckHandle) CheckAction() bool{
	return true
}

type HealthCheckHandle struct {
}

func NewHealthCheckHandle() *HealthCheckHandle{
	return  &HealthCheckHandle{
	}
}

func ParseOption(input string) KickHandle {
    arrays := strings.Split(input, " ")
    if len(arrays) < 2 {
    	return nil
	}
	var handle KickHandle = nil
	if arrays[1] == "agent" {
		handle = NewAgentHandle(arrays[2],arrays[3:])
	} else if arrays[1] == "task" {
		handle = NewTaskHandle(arrays[2],arrays[3:])
	} else if arrays[1] == "session" {
		handle = NewSessionHandle(arrays[2],arrays[3:])
	} else if arrays[1] == "noop" {
		handle = NewHealthCheckHandle()
	} else if arrays[1] == "file" {
		handle = NewFileHandle(arrays[2],arrays[3:])
	} else if arrays[1] == "status" {
		handle = NewStatusHandle(arrays[2], arrays[3:])
	}

	return handle
}