package kickvmhandle

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKickVm(t *testing.T) {
	handle := ParseOption("kick_vm agent stop")
	action := handle.CheckAction()
	assert.Equal(t, action, true)

	handle = ParseOption("kick_vm task run t-xx")
	action = handle.CheckAction()
	assert.Equal(t, action, true)

	handle = ParseOption("kick_vm agent1 stop")
	assert.Equal(t, handle, nil)

	handle = ParseOption("kick_vm agent stop1")
	action = handle.CheckAction()
	assert.Equal(t, action, false)


	handle = ParseOption("kick_vm agent stop")
	action = handle.CheckAction()
	assert.Equal(t, action, true)

	handle = ParseOption("kick_vm noop")
	action = handle.CheckAction()
	assert.Equal(t, action, true)
}