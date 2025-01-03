package taskmanager

import (
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/commander/taskerrors"
	"github.com/aliyun/aliyun_assist_client/commander/taskinterface"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/container"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	container_errors "github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
)

type taskManager struct {
	taskMap_ sync.Map

	// If task is in the PHASE_PRECHECKED or PHASE_EXITED phase for a long time,
	// it needs to be cleared from taskMap_
	taskExpiration int // second
}

var taskmanager *taskManager

func GetTaskManager() *taskManager {
	if taskmanager == nil {
		taskmanager = &taskManager{
			taskMap_:       sync.Map{},
			taskExpiration: 60, // second
		}
	}
	return taskmanager
}

func (tm *taskManager) NewTask(submissionId string, commandType string, timeout int, workingDir string, username string, annotation map[string]string) (taskinterface.Task, *taskerrors.TaskError) {
	var processer model.TaskProcessor
	var err error
	if annotation["containerId"] == "" && annotation["containerName"] == "" {
		containerError := container_errors.NewContainerNotFoundError()
		return nil, taskerrors.NewTaskError(containerError.ErrorCode, containerError.ErrorSubCode, containerError.ErrorMessage)
	}
	log.GetLogger().Infof("Receive task")
	processer, err = container.DetectContainerProcessor(&model.ContainerCommandOptions{
		SubmissionID:  submissionId,
		ContainerId:   annotation["containerId"],
		ContainerName: annotation["containerName"],
		CommandType:   commandType,
		Timeout:       timeout,

		WorkingDirectory: workingDir,
		Username:         username,
	})
	if err != nil {
		if containerErr, ok := err.(*container_errors.TaskError); ok {
			return nil, taskerrors.NewTaskError(containerErr.ErrorCode, containerErr.ErrorSubCode, containerErr.ErrorMessage)
		}
		return nil, taskerrors.NewGeneralError(err)
	}
	task := &Task{
		SubmissionId: submissionId,
		taskInfo: TaskInfo{
			ContainerId:   annotation["containerId"],
			ContainerName: annotation["containerName"],
			CommandType:   commandType,
			Timeout:       timeout,
			WorkingDir:    workingDir,
			Username:      username,
		},
		processer: processer,
		logger: log.GetLogger().WithField(
			"submissionId", submissionId,
		),
		extraLubanParams: processer.ExtraLubanParams(),
	}
	if _, loaded := tm.taskMap_.LoadOrStore(submissionId, task); loaded {
		return nil, taskerrors.NewSubmissionDuplicateError()
	}
	return task, nil
}

// LoadTask return task by submissionId
func (tm *taskManager) LoadTask(submissionId string) (taskinterface.Task, *taskerrors.TaskError) {
	if value, ok := tm.taskMap_.Load(submissionId); !ok {
		containerError := container_errors.NewSubmissionInvalidError("SubmissionID not exist")
		return nil, taskerrors.NewTaskError(containerError.ErrorCode, containerError.ErrorSubCode, containerError.ErrorMessage)
	} else {
		task, ok := value.(*Task)
		if !ok {
			return nil, taskerrors.NewSubmissionTypeInvalidError()
		}
		return task, nil
	}
}

// TaskCount returns live task count in TaskManager
func (tm *taskManager) TaskCount() int {
	count := 0
	deleteSubmissionList := []string{}
	tm.taskMap_.Range(func(key, value any) bool {
		if task, ok := value.(*Task); ok {
			if (task.phase == PHASE_PRECHECKED || task.phase == PHASE_EXITED) &&
				time.Since(task.phaseStartTime) > time.Duration(tm.taskExpiration)*time.Second {
				deleteSubmissionList = append(deleteSubmissionList, task.SubmissionId)
			} else {
				count += 1
			}
		}
		return true
	})
	for _, submissionId := range deleteSubmissionList {
		tm.taskMap_.Delete(submissionId)
	}
	return count
}

// PreCheckTask precheck task by submissionId
func (tm *taskManager) PreCheckTask(submissionId string) *taskerrors.TaskError {
	if taskValue, ok := tm.taskMap_.Load(submissionId); !ok {

		return taskerrors.NewSubmissionNotExistsError()
	} else {
		if task, ok := taskValue.(*Task); ok {
			return task.PreCheck()
			// implement more business code here
		} else {
			return taskerrors.NewSubmissionTypeInvalidError()
		}

	}
}

// PrepareTask prepare task with content by submissionId
func (tm *taskManager) PrepareTask(submissionId string, content string) *taskerrors.TaskError {
	if taskValue, ok := tm.taskMap_.Load(submissionId); !ok {
		return taskerrors.NewSubmissionNotExistsError()
	} else {
		if task, ok := taskValue.(*Task); ok {
			return task.Prepare(content)
			// implement more business code here
		} else {
			return taskerrors.NewSubmissionTypeInvalidError()
		}

	}
}

// Run will run task by submissionId;
// RunTask rely on inner task.run, run can be sync or async
func (tm *taskManager) RunTask(submissionId string) *taskerrors.TaskError {
	if taskValue, ok := tm.taskMap_.Load(submissionId); !ok {
		return taskerrors.NewSubmissionNotExistsError()
	} else {
		if task, ok := taskValue.(*Task); ok {
			go task.Run()
			// implement more business code here
			return nil
		} else {
			return taskerrors.NewSubmissionTypeInvalidError()
		}

	}
}

// Cancel cancel task by submissionId
func (tm *taskManager) CancelTask(submissionId string) *taskerrors.TaskError {
	if taskValue, ok := tm.taskMap_.Load(submissionId); !ok {
		return taskerrors.NewSubmissionNotExistsError()
	} else {
		if task, ok := taskValue.(*Task); ok {
			return task.Cancel()
		} else {
			return taskerrors.NewSubmissionTypeInvalidError()
		}

	}
}

// Dispose dispose task by submissionId
func (tm *taskManager) DisposeTask(submissionId string) *taskerrors.TaskError {
	if taskValue, ok := tm.taskMap_.Load(submissionId); !ok {
		return taskerrors.NewSubmissionNotExistsError()
	} else {
		if task, ok := taskValue.(*Task); ok {
			tm.taskMap_.Delete(submissionId)
			return task.Dispose()
		} else {
			return taskerrors.NewSubmissionTypeInvalidError()
		}

	}
}

func (tm *taskManager) CleanupTask(submissionId string) *taskerrors.TaskError {
	if taskValue, ok := tm.taskMap_.Load(submissionId); !ok {
		return taskerrors.NewSubmissionNotExistsError()
	} else {
		if task, ok := taskValue.(*Task); ok {
			tm.taskMap_.Delete(submissionId)
			return task.Cleanup()
		} else {
			return taskerrors.NewSubmissionTypeInvalidError()
		}

	}
}
