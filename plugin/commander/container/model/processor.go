package model

import (
	"io"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
)

type TaskProcessor interface {
	PreCheck() (invalidParameter string, err *taskerrors.TaskError)

	Prepare(commandContent string) *taskerrors.TaskError

	SyncRun(
		stdoutWriter io.Writer,
		stderrWriter io.Writer,
		stdinReader  io.Reader)  (exitCode int, status int, err error)
	Cancel() error

	Cleanup() error

	SideEffect() error

	ExtraLubanParams() string
}
