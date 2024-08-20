package processutil

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_pidAlive(t *testing.T) {
	assert.True(t, pidAlive(os.Getpid()))

	cmd := exec.Command(sleepCommand[0], sleepCommand...)
	assert.Nil(t, cmd.Start())

	assert.True(t, pidAlive(cmd.Process.Pid))
	assert.Nil(t, cmd.Process.Kill())
	cmd.Wait()
	assert.False(t, pidAlive(cmd.Process.Pid))
}