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
	processer.SyncRun("/tmp",
		commandName, nil, &stdoutWrite, &stderrWrite,  nil, nil, 30)

	assert.Contains(t,  stdoutWrite.String(), "/tmp")
}