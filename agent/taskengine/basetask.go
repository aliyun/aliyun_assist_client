package taskengine

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/commandermanager"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/commander"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/host"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/outputbuffer"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/parameters"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/scriptmanager"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
)

const (
	defaultQuoto    = 12000
	defaultQuotoPre = 6000
)

type FinishCallback func()
type ReportErrorCallback func(taskId string, repeat models.RunTaskRepeatType, errorCode, status string) (isTaskErr bool)

type Task struct {
	taskInfo         models.RunTaskInfo
	scheduleLocation *time.Location
	onFinish         FinishCallback
	onReportError 	 ReportErrorCallback

	processer               models.TaskProcessor
	startTime               time.Time
	endTime                 time.Time
	monotonicStartTimestamp int64
	monotonicEndTimestamp   int64
	exit_code               int
	canceled                bool
	droped                  int
	cancelMut               sync.Mutex

	disableOutputRingbuffer bool
	outputRingbuffer        outputbuffer.OutputBuffer
	output                  bytes.Buffer
	data_sended             uint32
}

func NewTask(taskInfo models.RunTaskInfo, scheduleLocation *time.Location, onFinish FinishCallback, onReportError ReportErrorCallback) (*Task, error) {
	timeout, err := strconv.Atoi(taskInfo.TimeOut)
	if err != nil {
		timeout = 3600
	}

	var processor models.TaskProcessor
	if taskInfo.ContainerId != "" || taskInfo.ContainerName != "" {
		annotation := map[string]string{
			"containerId":   taskInfo.ContainerId,
			"containerName": taskInfo.ContainerName,
		}
		processor, err = commander.NewCommanderProcessor(taskInfo, timeout, annotation, commandermanager.ContainerCommanderName)
		if err != nil {
			return nil, err
		}
	} else {
		processor = &host.HostProcessor{
			TaskId:        taskInfo.TaskId,
			InvokeVersion: taskInfo.InvokeVersion,
			CommandType:   taskInfo.CommandType,
			Repeat:        taskInfo.Repeat,
			Timeout:       timeout,

			CommandName:         taskInfo.CommandName,
			WorkingDirectory:    taskInfo.WorkingDir,
			Username:            taskInfo.Username,
			WindowsUserPassword: taskInfo.Password,
			TerminationMode:     taskInfo.TerminationMode,
		}
	}

	task := &Task{
		taskInfo:                taskInfo,
		scheduleLocation:        scheduleLocation,
		onFinish:                onFinish,
		onReportError:           onReportError,
		processer:               processor,
		canceled:                false,
		droped:                  0,
		disableOutputRingbuffer: flagging.IsTaskOutputRingbufferDisabled(),
	}

	return task, nil
}

func tryRead(stdouterrWrite io.Reader, out *bytes.Buffer) {
	buf_stdout := make([]byte, 2048)
	n, _ := stdouterrWrite.Read(buf_stdout)
	out.Write(buf_stdout[:n])
}
func tryReadAll(stdouterrWrite io.Reader, out *bytes.Buffer) {
	for {
		buf_stdout := make([]byte, 2048)
		n, _ := stdouterrWrite.Read(buf_stdout)
		out.Write(buf_stdout[:n])
		if n == 0 {
			break
		}
	}
}

func (task *Task) PreCheck(reportVerified bool) error {
	// Reuse specified logger across whole task pre-checking phase
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        task.taskInfo.TaskId,
		"InvokeVersion": task.taskInfo.InvokeVersion,
		"Phase":         "Pre-checking",
	})

	if task.taskInfo.CommandType != "RunBatScript" &&
		task.taskInfo.CommandType != "RunPowerShellScript" &&
		task.taskInfo.CommandType != "RunShellScript" {
		task.SendInvalidTask("TypeInvalid", fmt.Sprintf("TypeInvalid_%s", task.taskInfo.CommandType))
		err := fmt.Errorf("Invalid command type: %s", task.taskInfo.CommandType)
		taskLogger.Errorln("TypeInvalid", err.Error())
		return err
	}

	if _, err := base64.StdEncoding.DecodeString(task.taskInfo.Content); err != nil {
		task.SendInvalidTask("CommandContentInvalid", err.Error())
		wrapErr := fmt.Errorf("Invalid command content: decode error: %w", err)
		taskLogger.Errorln("CommandContentInvalid", wrapErr.Error())
		return wrapErr
	}

	if invalidParameter, err := task.processer.PreCheck(); err != nil {
		if validationErr, ok := err.(*taskerrors.CommanderError); ok {
			task.SendInvalidTask(validationErr.SubCode, validationErr.Error())
		} else if validationErr, ok := err.(taskerrors.NormalizedValidationError); ok {
			task.SendInvalidTask(validationErr.Param(), validationErr.Value())
		} else if settingErr, ok := err.(taskerrors.InvalidSettingError); ok {
			task.SendInvalidTask(invalidParameter, fmt.Sprintf("%s: %v", settingErr.ShortMessage(), settingErr.Unwrap()))
		} else {
			task.SendInvalidTask(invalidParameter, err.Error())
		}
		taskLogger.WithError(err).Errorf("Invalid parameter \"%s\" for invocation", invalidParameter)
		return err
	}

	if reportVerified == true {
		if err := task.sendTaskVerified(); err != nil {
			return err
		}
	}
	return nil
}

func (task *Task) Run() (taskerrors.ErrorCode, error) {
	if err := task.PreCheck(false); err != nil {
		if taskError, ok := err.(taskerrors.ExecutionError); ok {
			return taskError.ErrCode(), taskError
		}
		return 0, err
	}

	// Reuse specified logger across whole task running phase
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":            task.taskInfo.TaskId,
		"InvokeVersion":     task.taskInfo.InvokeVersion,
		"Phase":             "Running",
		"disableRingbuffer": task.disableOutputRingbuffer,
	})
	taskLogger.Info("Run task")

	taskLogger.Info("Prepare script file of task")
	decodeBytes, err := base64.StdEncoding.DecodeString(task.taskInfo.Content)
	if err != nil {
		task.SendError("", taskerrors.WrapErrBase64DecodeFailed, fmt.Sprintf("Base64DecodeFailed: %s", err.Error()))
		return taskerrors.WrapErrBase64DecodeFailed, errors.New("decode error")
	}
	ScriptToDelete := false
	content := string(decodeBytes)
	if task.taskInfo.EnableParameter {
		content, err = parameters.ResolveBuiltinParameters(content, task.taskInfo.BuiltinParameters)
		if err != nil {
			if invalidErr, ok := err.(taskerrors.InvalidSettingError); ok {
				task.SendInvalidTask("InvalidEnvironmentParameter", invalidErr.ShortMessage())
			} else if taskErr, ok := err.(taskerrors.ExecutionError); ok {
				task.SendError("", taskErr.ErrCode(), taskErr.Error())
				return taskErr.ErrCode(), err
			} else {
				task.SendError("", taskerrors.WrapErrResolveEnvironmentParameterFailed, err.Error())
			}

			return taskerrors.WrapErrResolveEnvironmentParameterFailed, err
		}

		if strings.Contains(content, "oos-secret") {
			ScriptToDelete = true
		}
		content, err = util.ReplaceAllParameterStore(content)
		if err != nil {
			task.SendInvalidTask(err.Error(), content)
			return 0, errors.New("ReplaceAllParameterStore error")
		}
	}

	switch task.taskInfo.CommandType {
	case "RunBatScript":
		content = "@echo off\r\n" + content

		fallthrough
	case "RunPowerShellScript":
		if !flagging.IsNormalizingCRLFDisabled() {
			content = scriptmanager.NormalizeCRLF(content)
		}
	}
	content = langutil.UTF8ToLocal(content)

	if err := task.processer.Prepare(content); err != nil {
		taskLogger.WithError(err).Errorln("Failed to prepare command process")
		if commErr, ok := err.(*taskerrors.CommanderError); ok {
			task.SendError("", taskerrors.Stringer(commErr.SubCode), commErr.Error())
			return taskerrors.WrapGeneralError, err
		} else if validationErr, ok := err.(taskerrors.NormalizedValidationError); ok {
			task.SendInvalidTask(validationErr.Param(), validationErr.Value())
			return taskerrors.WrapGeneralError, err
		} else if taskErr, ok := err.(taskerrors.ExecutionError); ok {
			task.SendError("", taskErr.ErrCode(), taskErr.Error())
			return taskErr.ErrCode(), err
		} else {
			return taskerrors.WrapGeneralError, err
		}

	}

	taskLogger.Info("Prepare command process")
	var stdouterrWriter io.ReadWriter
	if task.disableOutputRingbuffer {
		stdouterrWriter = &process.SafeBuffer{}
	} else {
		totalQuoto := task.taskInfo.Output.LogQuota
		if totalQuoto < defaultQuoto {
			totalQuoto = defaultQuoto
		}
		var err error
		stdouterrWriter, err = task.outputRingbuffer.Init(defaultQuotoPre, totalQuoto-defaultQuotoPre)
		if err != nil {
			taskLogger.Error("create pipe failed: ", err)
			taskError := taskerrors.NewCreatePipeError(err)
			task.SendError("", taskError.ErrCode(), taskError.Error())
			return taskError.ErrCode(), err
		}
	}

	task.startTime = time.Now()
	task.monotonicStartTimestamp = timetool.ToAccurateTime(task.startTime.Local())
	if taskError := task.sendTaskStart(); taskError != nil {
		taskLogger.WithError(taskError).Error("Send starting event failed")
		return taskError.ErrCode(), taskError
	}
	taskLogger.Infof("Sent starting event")

	// Replace variable representing states with context and channel operation,
	// to replace dangerous state tranfering operation with straightforward
	// message passing action.
	ctx, stopSendRunning := context.WithCancel(context.Background())
	stoppedSendRunning := make(chan struct{}, 1)
	go func(ctx context.Context, stoppedSendRunning chan<- struct{}) {
		defer close(stoppedSendRunning)
		task.data_sended = 0
		// Running output is not needed to be reported during invocation of
		// periodic tasks. But stoppedSendRunning channel is still needed to be
		// closed correctly.
		if task.taskInfo.Cronat != "" {
			return
		}

		intervalMs := task.taskInfo.Output.Interval
		if intervalMs < 1000 {
			intervalMs = 1000
		}
		ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
		lastReportOutputTime := time.Now()
		defer ticker.Stop()
		for {
			// serve the stop signal from context channel with higher priority
			select {
			case <-ctx.Done():
				return
			default:
				// fallthrough to the next select
			}

			select {
			case <-ticker.C:
				if task.disableOutputRingbuffer {
					if atomic.LoadUint32(&task.data_sended) > defaultQuotoPre {
						tryRead(stdouterrWriter, &task.output)
						if reported := task.sendRunningOutput("", lastReportOutputTime); reported {
							lastReportOutputTime = time.Now()
						}
						taskLogger.Infof("Running output sent: %d bytes, just report running no output sent", atomic.LoadUint32(&task.data_sended))
					} else {
						var running_output bytes.Buffer
						tryRead(stdouterrWriter, &running_output)
						if reported := task.sendRunningOutput(running_output.String(), lastReportOutputTime); reported {
							lastReportOutputTime = time.Now()
						}
						atomic.AddUint32(&task.data_sended, uint32(running_output.Len()))
						taskLogger.Infof("Running output sent: %d bytes", atomic.LoadUint32(&task.data_sended))
					}
				} else {
					outputPre := task.outputRingbuffer.ReadPre()
					if reported := task.sendRunningOutput(string(outputPre), lastReportOutputTime); reported {
						lastReportOutputTime = time.Now()
					}
					if len(outputPre) > 0 {
						atomic.AddUint32(&task.data_sended, uint32(len(outputPre)))
						taskLogger.Infof("Running output sent: %d bytes", atomic.LoadUint32(&task.data_sended))
					} else {
						taskLogger.Infof("Running output sent: %d bytes, just report running no output sent", atomic.LoadUint32(&task.data_sended))
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}(ctx, stoppedSendRunning)

	taskLogger.Info("Start command process")
	var status int
	task.exit_code, status, err = task.processer.SyncRun(stdouterrWriter, stdouterrWriter, nil)
	if status == process.Success {
		taskLogger.WithFields(logrus.Fields{
			"exitcode":   task.exit_code,
			"extraError": err,
		}).Info("Finished command process")
	} else if status == process.Timeout {
		taskLogger.WithFields(logrus.Fields{
			"attchedError": err,
		}).Info("Terminated command process due to timeout")
	} else if status == process.Fail {
		taskLogger.WithError(err).Info("Failed command process")
	} else {
		taskLogger.WithFields(logrus.Fields{
			"exitcode":     task.exit_code,
			"status":       status,
			"attchedError": err,
		}).Warn("Ended command process with unexpected status")
	}

	// That is, send stopping message to the goroutine sending running output
	stopSendRunning()
	// Wait for the goroutine sending running output to exit
	<-stoppedSendRunning
	task.endTime = time.Now()
	task.monotonicEndTimestamp = timetool.ToAccurateTime(timetool.ToStableElapsedTime(task.endTime, task.startTime).Local())

	var postOutput string
	if task.disableOutputRingbuffer {
		tryReadAll(stdouterrWriter, &task.output)
		postOutput = task.getReportString(task.output)
	} else {
		task.droped = task.outputRingbuffer.Dropped()
		postOutput = string(task.outputRingbuffer.ReadAll())
	}

	if status == process.Fail {
		if err == nil {
			task.sendOutput("failed", postOutput)
		} else if executionErr, ok := err.(taskerrors.NormalizedExecutionError); ok {
			task.SendError(postOutput, taskerrors.Stringer(executionErr.Code()), executionErr.Description())
		} else if taskErr, ok := err.(taskerrors.ExecutionError); ok {
			task.SendError(postOutput, taskErr.ErrCode(), taskErr.Error())
		} else {
			task.SendError(postOutput, taskerrors.WrapErrExecuteScriptFailed, fmt.Sprintf("ExecuteScriptFailed: %s", err.Error()))
		}
	} else if status == process.Timeout {
		task.sendOutput("timeout", postOutput)
	} else {
		if task.IsCancled() == false {
			task.sendOutput("finished", postOutput)
		}
	}
	endTaskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId":        task.taskInfo.TaskId,
		"InvokeVersion": task.taskInfo.InvokeVersion,
		"Phase":         "Ending",
	})
	endTaskLogger.Info("Sent final output and state")

	if task.disableOutputRingbuffer {
		task.output.Reset()
		endTaskLogger.Info("Clean task output")
	} else {
		if err := task.outputRingbuffer.Uninit(); err != nil {
			endTaskLogger.Error("Task outputbuffer err: ", err)
		} else {
			endTaskLogger.Info("Clean task output")
		}
	}

	// Perform cleanup actions after task finished
	if err := task.processer.Cleanup(ScriptToDelete); err != nil {
		endTaskLogger.WithError(err).Errorln("Failed to cleanup after command finished")
	}

	// Perform instructed poweroff/reboot action after task finished
	if err := task.processer.SideEffect(); err != nil {
		endTaskLogger.WithError(err).Errorln("Failed to apply side-effect of command after finished")
	}

	return 0, nil
}

func (task *Task) sendTaskVerified() error {
	queryParams := fmt.Sprintf("?taskId=%s&invokeVersion=%d",
		task.taskInfo.TaskId, task.taskInfo.InvokeVersion)
	url := util.GetVerifiedTaskService() + queryParams
	url += task.processer.ExtraLubanParams()
	content, _ := util.HttpPost(url, "", "text")
	if resp := parseTaskReportResp(content); resp != nil && resp.ErrorCode != "" {
		log.GetLogger().WithFields(logrus.Fields{
			"taskId":            task.taskInfo.TaskId,
			"invocationVersion": task.taskInfo.InvokeVersion,
		}).Errorf("Receive errorCode[%s] and status[%s] in response", resp.ErrorCode, resp.Status)
		if task.onReportError != nil && task.onReportError(task.taskInfo.TaskId, task.taskInfo.Repeat, resp.ErrorCode, resp.Status) {
			err := fmt.Errorf("Server response error[%s] and status[%s] when send task verified", resp.ErrorCode, resp.Status)
			taskError := taskerrors.NewServerResponseError(err)
			return taskError
		}
	}
	return nil
}

func (task *Task) sendTaskStart() taskerrors.ExecutionError {
	if task.taskInfo.Output.SendStart == false {
		return nil
	}
	url := util.GetRunningOutputService()
	url += fmt.Sprintf("?taskId=%s&invokeVersion=%d&start=%s",
		task.taskInfo.TaskId, task.taskInfo.InvokeVersion, strconv.FormatInt(task.monotonicStartTimestamp, 10))
	url += task.wallClockQueryParams()
	url += task.processer.ExtraLubanParams()

	content, _ := util.HttpPost(url, "", "text")
	if resp := parseTaskReportResp(content); resp != nil && resp.ErrorCode != "" {
		log.GetLogger().WithFields(logrus.Fields{
			"taskId":            task.taskInfo.TaskId,
			"invocationVersion": task.taskInfo.InvokeVersion,
		}).Errorf("Receive errorCode[%s] and status[%s] in response", resp.ErrorCode, resp.Status)
		if task.onReportError != nil && task.onReportError(task.taskInfo.TaskId, task.taskInfo.Repeat, resp.ErrorCode, resp.Status) {
			err := fmt.Errorf("Server response error[%s] when send task start", resp.ErrorCode)
			taskError := taskerrors.NewServerResponseError(err)
			return taskError
		}
	}
	return nil
}

func (task *Task) SendInvalidTask(param string, value string) {
	content, _ := reportInvalidTask(task.taskInfo.TaskId, task.taskInfo.InvokeVersion, param, value, task.processer.ExtraLubanParams())
	if resp := parseTaskReportResp(content); resp != nil && resp.ErrorCode != "" {
		log.GetLogger().WithFields(logrus.Fields{
			"taskId":            task.taskInfo.TaskId,
			"invocationVersion": task.taskInfo.InvokeVersion,
		}).Errorf("Receive errorCode[%s] and status[%s] in response", resp.ErrorCode, resp.Status)
		if task.onReportError != nil {
			task.onReportError(task.taskInfo.TaskId, task.taskInfo.Repeat, resp.ErrorCode, resp.Status)
		}
	}
}

func (task *Task) sendOutput(status string, output string) {
	output = langutil.LocalToUTF8(output)

	var url string
	if status == "finished" {
		url = util.GetFinishOutputService()
	} else if status == "timeout" {
		url = util.GetTimeoutOutputService()
	} else if status == "failed" {
		url = util.GetErrorOutputService()
	} else {
		return
	}

	url += fmt.Sprintf("?taskId=%s&invokeVersion=%d&start=%s&end=%s&exitCode=%s&dropped=%s",
		task.taskInfo.TaskId, task.taskInfo.InvokeVersion, strconv.FormatInt(task.monotonicStartTimestamp, 10),
		strconv.FormatInt(task.monotonicEndTimestamp, 10), strconv.Itoa(task.exit_code), strconv.Itoa(task.droped))
	url += task.wallClockQueryParams()
	url += task.processer.ExtraLubanParams()

	var err error
	var content string
	content, err = util.HttpPost(url, output, "text")

	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		content, err = util.HttpPost(url, output, "text")
	}

	if resp := parseTaskReportResp(content); resp != nil && resp.ErrorCode != "" {
		log.GetLogger().WithFields(logrus.Fields{
			"taskId":            task.taskInfo.TaskId,
			"invocationVersion": task.taskInfo.InvokeVersion,
		}).Errorf("Receive errorCode[%s] and status[%s] in response", resp.ErrorCode, resp.Status)
		if task.onReportError != nil {
			task.onReportError(task.taskInfo.TaskId, task.taskInfo.Repeat, resp.ErrorCode, resp.Status)
		}
	}

	if task.onFinish != nil {
		task.onFinish()
	}
}

func (task *Task) SendError(output string, errCode fmt.Stringer, errDesc string) {
	safelyTruncatedErrDesc := langutil.SafeTruncateStringInBytes(errDesc, 255)
	escapedErrDesc := url.QueryEscape(safelyTruncatedErrDesc)
	queryString := fmt.Sprintf("?taskId=%s&invokeVersion=%d&start=%d&end=%d&exitCode=%d&dropped=%d&errCode=%s&errDesc=%s",
		task.taskInfo.TaskId, task.taskInfo.InvokeVersion, task.monotonicStartTimestamp,
		task.monotonicEndTimestamp, task.exit_code, task.droped, errCode.String(), escapedErrDesc)
	queryString += task.wallClockQueryParams()
	queryString += task.processer.ExtraLubanParams()

	requestURL := util.GetErrorOutputService() + queryString

	if len(output) > 0 {
		output = langutil.LocalToUTF8(output)
	}

	content, err := util.HttpPost(requestURL, output, "text")
	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		content, err = util.HttpPost(requestURL, output, "text")
	}
	if resp := parseTaskReportResp(content); resp != nil && resp.ErrorCode != "" {
		log.GetLogger().WithFields(logrus.Fields{
			"taskId":            task.taskInfo.TaskId,
			"invocationVersion": task.taskInfo.InvokeVersion,
		}).Errorf("Receive errorCode[%s] and status[%s] in response", resp.ErrorCode, resp.Status)
		if task.onReportError != nil {
			task.onReportError(task.taskInfo.TaskId, task.taskInfo.Repeat, resp.ErrorCode, resp.Status)
		}
	}
}

// Cancel the task invocation. If quietly is false, notify server the task is canceled.
func (task *Task) Cancel(quietly bool) error {
	task.cancelMut.Lock()
	defer task.cancelMut.Unlock()
	if task.canceled {
		return nil
	}
	task.canceled = true
	// Consistent with C++ version, end time of canceled task is set to the time
	// of cancel operation
	task.endTime = time.Now()
	if task.startTime.IsZero() {
		task.monotonicEndTimestamp = timetool.ToAccurateTime(task.endTime.Local())
	} else {
		task.monotonicEndTimestamp = timetool.ToAccurateTime(timetool.ToStableElapsedTime(task.endTime, task.startTime).Local())
	}
	cancelErr := task.processer.Cancel()
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"taskId":        task.taskInfo.TaskId,
		"invokeVersion": task.taskInfo.InvokeVersion,
	})
	if cancelErr == nil {
		taskLogger.Info("Task canceled")
	} else {
		taskLogger.WithError(cancelErr).Error("Task canceled failed")
	}

	if !quietly {
		stopResult := stopReasonKilled
		if cancelErr != nil {
			stopResult = stopFailed
		}
		var output string
		if task.disableOutputRingbuffer {
			output = task.getReportString(task.output)
		} else {
			task.droped = task.outputRingbuffer.Dropped()
			output = string(task.outputRingbuffer.ReadAll())
		}
		sendStoppedOutput(task.taskInfo.TaskId, task.taskInfo.InvokeVersion,
			task.monotonicStartTimestamp, task.monotonicEndTimestamp, task.exit_code,
			task.droped, output, stopResult, cancelErr)
	}
	return cancelErr
}

func (task *Task) getReportString(output bytes.Buffer) string {
	var report_string string
	quoto := task.taskInfo.Output.LogQuota
	if quoto < defaultQuoto {
		quoto = defaultQuoto
	}
	data_sended := atomic.LoadUint32(&task.data_sended)
	if output.Len() <= quoto-int(data_sended) {
		report_string = output.String()
	} else {
		bytes_data := output.Bytes()
		task.droped = output.Len() - (quoto - int(data_sended))
		report_string = string(bytes_data[task.droped:])
	}
	return report_string
}

func (task *Task) sendRunningOutput(data string, lastReportTime time.Time) bool {
	if len(data) == 0 && task.taskInfo.Output.SkipEmpty && time.Since(lastReportTime) < time.Minute {
		return false
	}
	url := util.GetRunningOutputService()
	url += fmt.Sprintf("?taskId=%s&invokeVersion=%d&start=%s",
		task.taskInfo.TaskId, task.taskInfo.InvokeVersion, strconv.FormatInt(task.monotonicStartTimestamp, 10))
	url += task.wallClockQueryParams()
	url += task.processer.ExtraLubanParams()

	data = langutil.LocalToUTF8(data)
	if content, err := util.HttpPost(url, data, "text"); err == nil {
		if resp := parseTaskReportResp(content); resp != nil && resp.ErrorCode != "" {
			log.GetLogger().WithFields(logrus.Fields{
				"taskId":            task.taskInfo.TaskId,
				"invocationVersion": task.taskInfo.InvokeVersion,
			}).Errorf("Receive errorCode[%s] and status[%s] in response", resp.ErrorCode, resp.Status)
			if task.onReportError != nil && task.onReportError(task.taskInfo.TaskId, task.taskInfo.Repeat, resp.ErrorCode, resp.Status) {
				// The task has started running, we need cancel it if the 
				// backend server tells us this task has an error.
				task.Cancel(true)
			}
		}
	}
	return true
}

func (task *Task) IsCancled() bool {
	task.cancelMut.Lock()
	defer task.cancelMut.Unlock()
	return task.canceled
}

// ResetCancel reset the canceled flag
func (task *Task) ResetCancel() {
	task.cancelMut.Lock()
	defer task.cancelMut.Unlock()
	task.canceled = false
}

// Generate additional querystring parameters: Unix timestamp of wall clock for
// cron/rate tasks, and timezone name of schedule clock for only cron tasks
func (task *Task) wallClockQueryParams() string {
	switch task.taskInfo.Repeat {
	case models.RunTaskRate:
		return fmt.Sprintf("&currentTime=%d", timetool.GetAccurateTime())
	case models.RunTaskCron:
		if task.scheduleLocation != nil {
			// NOTE: The time stdlib of golang hopelessly mixes nil pointer and
			// pointer to pre-defined utcLoc for some Location methods, e.g.,
			// String(). That is, even `*time.Location(nil).String()` would
			// return "UTC" instead of just panic. Be careful with this!!!
			escapedTimezoneName := url.QueryEscape(task.scheduleLocation.String())
			locatedNow := time.Now().In(task.scheduleLocation)
			_, currentOffsetFromUTC := locatedNow.Zone()
			return fmt.Sprintf("&currentTime=%d&offset=%d&timeZone=%s", timetool.ToAccurateTime(locatedNow), currentOffsetFromUTC, escapedTimezoneName)
		} else {
			currentTime, currentOffsetFromUTC, timezoneName := timetool.NowWithTimezoneName()
			escapedTimezoneName := url.QueryEscape(timezoneName)
			return fmt.Sprintf("&currentTime=%d&offset=%d&timeZone=%s", timetool.ToAccurateTime(currentTime), currentOffsetFromUTC, escapedTimezoneName)
		}
	}

	return ""
}
