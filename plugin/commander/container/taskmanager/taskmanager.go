package taskmanager

import (
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/container"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/client"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type Task struct {
	SubmissionId string
	taskInfo     TaskInfo
	processer    model.TaskProcessor
	canceled     atomic.Bool
	sendIndex    atomic.Int32
	exit_code    int

	logger           logrus.FieldLogger
	extraLubanParams string

	// Record task phase and when current phase began.
	// phase and phaseModifyTime are used to clear expired tasks.
	phase          int
	phaseStartTime time.Time
}

type TaskInfo struct {
	ContainerId   string
	ContainerName string
	CommandType   string
	Timeout       int
	WorkingDir    string
	Username      string
}

const (
	// prechecked
	PHASE_PRECHECKED = iota
	// running
	PHASE_RUNNING
	// process exited or task is canceled, but Dispose not called
	PHASE_EXITED
)

var (
	taskMap_ sync.Map

	// If task is in the PHASE_PRECHECKED or PHASE_EXITED phase for a long time,
	// it needs to be cleared from taskMap_
	taskExpiration = 60 // second
)

func NewTask(submissionId, containerId, containerName, commandType, workingDir, username string, timeout int) (*Task, error) {
	var processer model.TaskProcessor
	var err error
	if containerId == "" && containerName == "" {
		return nil, taskerrors.NewContainerNotFoundError()
	}
	processer, err = container.DetectContainerProcessor(&model.ContainerCommandOptions{
		SubmissionID:  submissionId,
		ContainerId:   containerId,
		ContainerName: containerName,
		CommandType:   commandType,
		Timeout:       timeout,

		WorkingDirectory: workingDir,
		Username:         username,
	})
	if err != nil {
		return nil, err
	}
	task := &Task{
		SubmissionId: submissionId,
		taskInfo: TaskInfo{
			ContainerId:   containerId,
			ContainerName: containerName,
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
	return task, nil
}

// PreCheck call processer.PreCheck  check if task is valid.
func (task *Task) PreCheck() *taskerrors.TaskError {
	taskLogger := task.logger.WithField("Phase", "Pre-checking")
	if _, taskerror := task.processer.PreCheck(); taskerror != nil {
		taskLogger.WithError(taskerror).Errorf("Task precheck failed.")
		return taskerror
	}

	if err := storeTask(task.SubmissionId, task); err != nil {
		return err
	}
	// Task phase is precheked
	task.phase = PHASE_PRECHECKED
	task.phaseStartTime = time.Now()
	taskLogger.Infof("Store task: %+v", task.taskInfo)
	
	return nil
}

func (task *Task) Prepare(content string) *taskerrors.TaskError {
	taskLogger := task.logger.WithField("Phase", "Prepare")
	if err := task.processer.Prepare(content); err != nil {
		taskLogger.Error("Processer prepare failed: ", err)
		return err
	}
	return nil
}

func (task *Task) Run() {
	taskLogger := task.logger.WithField("Phase", "Running")
	stdouterrR, stdouterrW, err := os.Pipe()
	if err != nil {
		taskLogger.Error("Create pipe failed: ", err)
		taskError := taskerrors.NewCreatePipeError(err)
		task.sendError(taskError)
		return
	}

	// Task phase is running
	task.phase = PHASE_RUNNING
	task.phaseStartTime = time.Now()
	defer func() {
		// Task phase is exited
		task.phase = PHASE_EXITED
		task.phaseStartTime = time.Now()
	}()

	task.sendRunningOutput("")
	stoppedSendRunning := make(chan struct{}, 1)
	buf := make([]byte, 10240)
	go func() {
		defer close(stoppedSendRunning)
		for {
			stdouterrR.SetReadDeadline(time.Now().Add(time.Minute))
			n, err := stdouterrR.Read(buf)
			if n > 0 {
				task.sendRunningOutput(string(buf[:n]))
			} else {
				task.sendRunningOutput("")
			}
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
					taskLogger.Info("read stdouterr finished")
				} else {
					taskLogger.Error("read stdouterr failed: ", err)
				}
				return
			}
		}
	}()

	taskLogger.Info("Start command process")
	var status int
	var taskerror *taskerrors.TaskError
	var ok bool
	task.exit_code, status, err = task.processer.SyncRun(stdouterrW, stdouterrW, nil)
	if err != nil {
		if taskerror, ok = err.(*taskerrors.TaskError); !ok {
			taskerror = taskerrors.NewGeneralExecutionError(err)
		}
	}

	taskLogger.Infof("Tasks syncrun done, exitCode[%d] status[%d] taskerror: %v", task.exit_code, status, taskerror)
	if status == process.Success {
		taskLogger.WithFields(logrus.Fields{
			"exitcode":   task.exit_code,
			"extraError": taskerror,
		}).Info("Finished command process")
	} else if status == process.Timeout {
		taskLogger.WithFields(logrus.Fields{
			"attchedError": taskerror,
		}).Info("Terminated command process due to timeout")
	} else if status == process.Fail {
		taskLogger.WithError(taskerror).Info("Failed command process")
	} else {
		taskLogger.WithFields(logrus.Fields{
			"exitcode":     task.exit_code,
			"status":       status,
			"attchedError": taskerror,
		}).Warn("Ended command process with unexpected status")
	}

	// close stdouterrW make stdouterrR read goroutine return
	stdouterrW.Close()
	<-stoppedSendRunning
	stdouterrR.Close()

	task.sendOutput("", status, taskerror)
}

func (task *Task) Cancel() {
	if task.canceled.CompareAndSwap(false, true) {
		task.processer.Cancel()
	}
	// Task phase is exited
	task.phase = PHASE_EXITED
	task.phaseStartTime = time.Now()
}

func (task *Task) Cleanup() error {
	// actualy processer.Cleanup does nothing
	return task.processer.Cleanup()
}

// Dispose is the last step, delete task from taskMap_ in this step
func (task *Task) Dispose() error {
	deleteTask(task.SubmissionId)
	task.logger.Infof("Delete task for dispose: %+v", task.taskInfo)
	return task.processer.SideEffect()
}

func (task *Task) ExtraLubanParams() string {
	return task.extraLubanParams
}

func (task *Task) sendError(taskError *taskerrors.TaskError) {
	idx := task.sendIndex.Add(1) - 1
	// FinalizeSubmissionStatus: exitCode is valid
	// only when the taskStatus is process.Success.
	task.logger.Infof("send error: index[%d] err: %s", idx, taskError)
	if err := client.FinalizeSubmissionStatus(task.logger, int(idx), task.SubmissionId, "",
		0, process.Fail, taskError); err != nil {
		task.logger.Error("sendError failed: ", err)
	}
}

func (task *Task) sendOutput(output string, taskStatus int, taskError *taskerrors.TaskError) {
	idx := task.sendIndex.Add(1) - 1
	task.logger.Infof("send output: index[%d] len[%d]", idx, len(output))
	if err := client.FinalizeSubmissionStatus(task.logger, int(idx), task.SubmissionId, output,
		task.exit_code, taskStatus, taskError); err != nil {
		task.logger.Error("send output failed: ", err)
	}
}

func (task *Task) sendRunningOutput(output string) {
	idx := task.sendIndex.Add(1) - 1
	task.logger.Infof("send running output: index[%d] len[%d]", idx, len(output))
	if err := client.WriteSubmissionOutput(task.logger, int(idx), task.SubmissionId, output); err != nil {
		task.logger.Error("send running output failed: ", err)
	}
}

func LoadTask(submissionId string) (*Task, *taskerrors.TaskError) {
	if value, ok := taskMap_.Load(submissionId); !ok {
		return nil, taskerrors.NewSubmissionInvalidError("SubmissionID not exist")
	} else {
		task, ok := value.(*Task)
		if !ok {
			return nil, taskerrors.NewSubmissionInvalidError("Type invalid")
		}
		return task, nil
	}
}

func storeTask(submissionId string, task *Task) *taskerrors.TaskError {
	if _, ok := taskMap_.LoadOrStore(submissionId, task); ok {
		return taskerrors.NewSubmissionInvalidError("SubmissionID is duplicated")
	}
	return nil
}

func deleteTask(keyId string) {
	taskMap_.Delete(keyId)
}

// TaskCount delete expired task and return number of valid tasks
func TaskCount() int {
	count := 0
	deleteSubmissionList := []string{}
	taskMap_.Range(func(key, value any) bool {
		if task, ok := value.(*Task); ok {
			if (task.phase == PHASE_PRECHECKED || task.phase == PHASE_EXITED) &&
				time.Since(task.phaseStartTime) > time.Duration(taskExpiration)*time.Second {
				deleteSubmissionList = append(deleteSubmissionList, task.SubmissionId)
			} else {
				count += 1
			}
		}
		return true
	})
	for _, submissionId := range deleteSubmissionList {
		taskMap_.Delete(submissionId)
	}
	return count
}
