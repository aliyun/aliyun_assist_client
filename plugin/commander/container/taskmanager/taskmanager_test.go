package taskmanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadStoreDeleteTask(t *testing.T) {
	task := &Task{
		SubmissionId: "123",
	}
	err := storeTask(task.SubmissionId, task)
	assert.Nil(t, err)
	task, err = LoadTask(task.SubmissionId)
	assert.Nil(t, err)
	assert.NotEqual(t, nil, task)
}
