package models

import (
	"io"
)

type TaskProcessor interface {
	PreCheck() (invalidParameter string, err error)

	Prepare(commandContent string) error

	SyncRun(
		stdoutWriter io.Writer,
		stderrWriter io.Writer,
		stdinReader io.Reader) (exitCode int, status int, err error)
	Cancel() error

	Cleanup(removeScriptFile bool) error

	SideEffect() error

	ExtraLubanParams() string
}
