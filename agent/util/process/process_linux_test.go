package process

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPwdCommnad(t *testing.T) {
	var stdoutWrite bytes.Buffer
	var stderrWrite bytes.Buffer
	processer :=  ProcessCmd{}

	commandName := "pwd"
	code,_, err := processer.SyncRun("/tmp",
		commandName, nil, &stdoutWrite, &stderrWrite,  nil, 30)

	assert.Contains(t,  stdoutWrite.String(), "/tmp")
}