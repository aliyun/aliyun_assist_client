package process

import (
"bytes"
"fmt"
"github.com/stretchr/testify/assert"
"testing"
)

func TestWinCommnad(t *testing.T) {
	var stdoutWrite bytes.Buffer
	var stderrWrite bytes.Buffer
	processer :=  ProcessCmd{}

	commandName := "cmd"
	code,_, err := processer.SyncRun("",
		commandName, nil, &stdoutWrite, &stderrWrite,  nil, nil, 30)

	fmt.Println(code, err)
	assert.Contains(t,  stdoutWrite.String(), "Microsoft")
}