package taskinterface

import (
	"github.com/aliyun/aliyun_assist_client/commander/taskerrors"
)

// Task is an abstract interface for commander-side commander-agent ipc
type Task interface {
	// PreCheck checks run-time env
	PreCheck() *taskerrors.TaskError

	ExtraLubanParams() string

	// Use content to build input params for task execute;
	// If input params invalid, prepare returns error
	Prepare(content string) *taskerrors.TaskError

	// Task run based on PreCheck-params and Prepare-content
	// Task run can be sync or async
	Run()

	// Task Cancel, if nothing to do just return
	Cancel() *taskerrors.TaskError

	// Task Dispose
	Dispose() *taskerrors.TaskError
	Cleanup() *taskerrors.TaskError
}

// TaskManager is an abstract interface for commander-side commander-agent ipc
type TaskManager interface {
	// NewTask create task with input params
	NewTask(submissionId string, commandType string, timeout int, workingDir string, userName string, annotation map[string]string) (Task, *taskerrors.TaskError)

	// LoadTask return task by submissionId
	LoadTask(submissionId string) (Task, *taskerrors.TaskError)

	// TaskCount returns live task count in TaskManager
	TaskCount() int

	// PreCheckTask precheck task by submissionId
	PreCheckTask(submissionId string) *taskerrors.TaskError

	// PrepareTask prepare task with content by submissionId
	PrepareTask(submissionId string, content string) *taskerrors.TaskError

	// Run will run task by submissionId;
	// RunTask rely on inner task.run, run can be sync or async
	RunTask(submissionId string) *taskerrors.TaskError

	// Cancel cancel task by submissionId
	CancelTask(submissionId string) *taskerrors.TaskError

	// CleanupTask cleanup task by submissionId
	CleanupTask(submissionId string) *taskerrors.TaskError

	// Dispose dispose task by submissionId
	DisposeTask(submissionId string) *taskerrors.TaskError
}
