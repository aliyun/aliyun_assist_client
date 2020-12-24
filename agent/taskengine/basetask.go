package taskengine

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hectane/go-acl"
	"github.com/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/langutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
)

const (
	defaultQuoto    = 12000
	defaultQuotoPre = 6000
)

type Task struct {
	taskInfo    RunTaskInfo
	processer   process.ProcessCmd
	startTime   int64
	endTime     int64
	exit_code   int
	canceled    bool
	droped      int
	cancelMut   sync.Mutex
	output      bytes.Buffer
	data_sended uint32
}

func NewTask(taskInfo RunTaskInfo) *Task {
	task := &Task{
		taskInfo:  taskInfo,
		processer: process.ProcessCmd{},
		canceled:  false,
		droped:    0,
	}

	return task
}

type RunTaskInfo struct {
	InstanceId  string `json:"instanceId"`
	CommandType string `json:"type"`
	TaskId      string `json:"taskID"`
	CommandId   string `json:"commandId"`
	TimeOut     string `json:"timeOut"`
	CommandName string `json:"commandName"`
	Content     string `json:"commandContent"`
	WorkingDir  string `json:"workingDirectory"`
	Args        string `json:"args"`
	Cronat      string `json:"cron"`
	Username    string `json:"username"`
	Password    string `json:"windowsPasswordName"`
	Output      OutputInfo
}

type SendFileTaskInfo struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	Destination string `json:"destination"`
	Group       string `json:"group"`
	Mode        string `json:"mode"`
	Name        string `json:"name"`
	Overwrite   bool   `json:"overwrite"`
	Owner       string `json:"owner"`
	Signature   string `json:"signature"`
	TaskID      string `json:"taskID"`
	Timeout     int64  `json:"timeout"`
	Output      OutputInfo
}

type OutputInfo struct {
	Interval  int  `json:"interval"`
	LogQuota  int  `json:"logQuota"`
	SkipEmpty bool `json:"skipEmpty"`
	SendStart bool `json:"sendStart"`
}

// presetWrapErrorCode defines and MUST contain all error codes that will be reported
// as failure
type presetWrapErrorCode int
const (
	wrapErrGetScriptPathFailed presetWrapErrorCode = -(1 + iota)
	wrapErrUnknownCommandType
	wrapErrBase64DecodeFailed
	wrapErrSaveScriptFileFailed
	wrapErrSetExecutablePermissionFailed
	wrapErrSetWindowsPermissionFailed
	wrapErrExecuteScriptFailed
)

func (task *Task) buildPath(fileName string, content string) bool {
	if ret := util.CheckFileIsExist(fileName); ret == true {
		return false
	}

	if err := util.WriteStringToFile(fileName, content); err != nil {
		return false
	}
	return true
}

func tryRead(stdoutWrite, stderrWrite io.Reader, out *bytes.Buffer) {
	buf_stdout := make([]byte, 1024)
	n, _ := stdoutWrite.Read(buf_stdout)
	buf_stderr := make([]byte, 1024)
	m, _ := stderrWrite.Read(buf_stderr)
	out.Write(buf_stdout[:n])
	out.Write(buf_stderr[:m])
}

func tryReadAll(stdoutWrite, stderrWrite io.Reader, out *bytes.Buffer) {
	for {
		buf_stdout := make([]byte, 1024)
		n, _ := stdoutWrite.Read(buf_stdout)
		buf_stderr := make([]byte, 1024)
		m, _ := stderrWrite.Read(buf_stderr)
		out.Write(buf_stdout[:n])
		out.Write(buf_stderr[:m])
		if m == 0 && n == 0 {
			break
		}
	}
}

func (task *Task) Run() error {
	if len(task.taskInfo.Username) > 0 {
		if runtime.GOOS == "linux" {
			_, _, _, err := process.GetUserCredentials(task.taskInfo.Username)
			if err != nil {
				info := "UserInvalid_"+task.taskInfo.Username
				task.SendInvalidTask("UsernameOrPasswordInvalid", info)
				log.GetLogger().Errorln("UsernameOrPasswordInvalid", info)
				return err
			}
			return err
		} else if runtime.GOOS == "windows" {
			err := process.IsUserValid(task.taskInfo.Username, task.taskInfo.Password)
			if err != nil {
				info := "UsernameOrPasswordInvalid_"+task.taskInfo.Username
				task.SendInvalidTask(err.Error(), info)
				log.GetLogger().Errorln("UsernameOrPasswordInvalid", err.Error(), info)
				return err
			}
		}
	}

	// Reuse specified logger across whole task running phase
	taskLogger := log.GetLogger().WithFields(logrus.Fields{
		"TaskId": task.taskInfo.TaskId,
		"Phase":  "Running",
	})

	taskLogger.Info("Run task")
	taskLogger.Info("Prepare script file of task")
	var fileName string
	var err error
	if fileName, err = util.GetScriptPath(); err != nil {
		taskLogger.WithError(err).Errorln("script path is error")
		task.SendError("", wrapErrGetScriptPathFailed, err.Error())
		return err
	}

	cmdType := task.taskInfo.CommandType
	var cmdTypeName string
	if cmdType == "RunBatScript" {
		cmdTypeName = ".bat"
	} else if cmdType == "RunShellScript" {
		cmdTypeName = ".sh"
		if len(task.taskInfo.Username) > 0 {
			fileName = "/tmp"
		}
	} else if cmdType == "RunPowerShellScript" {
		cmdTypeName = ".ps1"
		task.processer.SyncRunSimple("powershell.exe", strings.Split("Set-ExecutionPolicy RemoteSigned", " "), 10)
	} else {
		taskLogger.Errorln("unkwown command type")
		task.SendError("", wrapErrUnknownCommandType, "unkwown command type")
		return errors.New("unkwown command type")
	}
	fileName = fileName + "/" + task.taskInfo.TaskId + cmdTypeName

	decodeBytes, err := base64.StdEncoding.DecodeString(task.taskInfo.Content)
	if err != nil {
		task.SendError("", wrapErrBase64DecodeFailed, err.Error())
		return errors.New("decode error")
	}

	content := string(decodeBytes)
	if cmdType == "RunBatScript" {
		content = "@echo off\r\n" + content
	}
	if G_IsWindows {
		if langutil.GetDefaultLang() != 0x409 {
			tmp, _ := langutil.Utf8ToGbk([]byte(content))
			content = string(tmp)
		}
	}

	writeScriptFileSuccess := task.buildPath(fileName, content)
	// Only non-periodic tasks need to check whether command script file exists.
	if !writeScriptFileSuccess && task.taskInfo.Cronat == "" {
		log.GetLogger().Errorln("build path is error")
		task.SendError("", wrapErrSaveScriptFileFailed, "build path is error")
		return errors.New("build path is error")
	}

	// Set executable permission bit of shell script file
	if cmdType == "RunShellScript" {
		if err := task.processer.SyncRunSimple("chmod", []string{"+x", fileName}, 10); err != nil {
			task.SendError("", wrapErrSetExecutablePermissionFailed, fmt.Sprintf("Failed to set executable permission of shell script: %s", err.Error()))
			taskLogger.WithError(err).Errorf("Failed to set executable permission of shell script")
		}
	} else {
		if len(task.taskInfo.Username) > 0 {
			if err := acl.Chmod(fileName, 0755); err != nil {
				task.SendError("", wrapErrSetWindowsPermissionFailed, fmt.Sprintf("Failed to set permission of script on Windows: %s", err.Error()))
				taskLogger.WithError(err).Errorf("Failed to set permission of script on Windows")
			}
		}
	}

	taskLogger.Info("Prepare command process")
	var stdoutWrite process.SafeBuffer
	var stderrWrite process.SafeBuffer
	timeout, err := strconv.Atoi(task.taskInfo.TimeOut)
	if err != nil {
		timeout = 3600
	}

	task.startTime = timetool.GetAccurateTime()
	args := make([]string, 2)
	if cmdType == "RunPowerShellScript" {
		args[0] = "-file"
		args[1] = fileName
		fileName = "powershell"
	} else if cmdType == "RunShellScript" {
		args[0] = "-c" // TODO: 兼容freebsd
		args[1] = fileName
		fileName = "sh"
	}

	// Replace variable+mutex with variable+atomic operation to remove dangerous
	// locking/unlocking operation
	var neverDirectWrite_Atomic_taskFinished uint32 = 0
	task.sendTaskStart()
	taskLogger.Infof("Sent starting event")

	go func() {
		task.data_sended = 0
		for {
			if task.taskInfo.Cronat != "" || atomic.LoadUint32(&task.data_sended) > defaultQuotoPre {
				break
			}
			var running_output bytes.Buffer
			tryRead(&stdoutWrite, &stderrWrite, &running_output)
			task.sendRunningOutput(running_output.String())
			atomic.AddUint32(&task.data_sended, uint32(running_output.Len()))
			taskLogger.Infof("Running output sent: %d bytes", atomic.LoadUint32(&task.data_sended))

			mseconds := task.taskInfo.Output.Interval
			if mseconds < 1000 {
				mseconds = 1000
			}
			time.Sleep(time.Duration(mseconds) * time.Millisecond)

			// Replace variable+mutex with variable+atomic operation to remove
			// dangerous locking/unlocking operation
			if atomic.LoadUint32(&neverDirectWrite_Atomic_taskFinished) == 1 {
				break
			}
		}
	}()

	taskLogger.Info("Start command process")
	var status int
	if len(task.taskInfo.Username) > 0 {
		task.processer.SetUserInfo(task.taskInfo.Username)
	}
    if len(task.taskInfo.Password) > 0 {
    	task.processer.SetPasswordInfo(task.taskInfo.Password)
	}

	task.exit_code, status, err = task.processer.SyncRun(task.taskInfo.WorkingDir,
		fileName, args,
		&stdoutWrite, &stderrWrite,
		nil, timeout)
	if status == process.Success {
		taskLogger.WithFields(logrus.Fields{
			"exitcode": task.exit_code,
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
			"exitcode": task.exit_code,
			"status": status,
			"attchedError": err,
		}).Warn("Ended command process with unexpected status")
	}

	// Replace variable+mutex with variable+atomic operation to remove dangerous
	// locking/unlocking operation
	atomic.StoreUint32(&neverDirectWrite_Atomic_taskFinished, 1)
	tryReadAll(&stdoutWrite, &stderrWrite, &task.output)

	task.endTime = timetool.GetAccurateTime()

	if status == process.Fail {
		if err == nil {
			task.sendOutput("failed", task.getReportString(task.output))
		} else {
			task.SendError(task.getReportString(task.output), wrapErrExecuteScriptFailed, err.Error())
		}
	} else if status == process.Timeout {
		task.sendOutput("timeout", task.getReportString(task.output))
	} else {
		if task.IsCancled() == false {
			task.sendOutput("finished", task.getReportString(task.output))
		}
	}
	taskLogger.Info("Sent final output and state")

	task.output.Reset()
	taskLogger.Info("Clean task output")

	return nil
}

func (task *Task) sendTaskStart() {
	if task.taskInfo.Output.SendStart == false {
		return
	}
	url := util.GetRunningOutputService()
	url += "?taskId=" + task.taskInfo.TaskId + "&start=" + strconv.FormatInt(task.startTime, 10)
	util.HttpPost(url, "", "text")
}

func (task *Task) SendInvalidTask(param string, value string) {
	url := util.GetInvalidTaskService()
	url += "?taskId=" + task.taskInfo.TaskId + "&param=" + param + "&value=" + value
	util.HttpPost(url, "", "text")
}

func (task *Task) sendOutput(status string, output string) {
	var url string
	if status == "finished" {
		url = util.GetFinishOutputService()
	} else if status == "timeout" {
		url = util.GetTimeoutOutputService()
	} else if status == "canceled" {
		url = util.GetStoppedOutputService()
	} else if status == "failed" {
		url = util.GetErrorOutputService()
	} else {
		return
	}

	url += "?taskId=" + task.taskInfo.TaskId + "&start=" + strconv.FormatInt(task.startTime, 10)
	url += "&end=" + strconv.FormatInt(task.endTime, 10) + "&exitCode=" + strconv.Itoa(task.exit_code) + "&dropped=" + strconv.Itoa(task.droped)
	// luban/api/v1/task/stopped API requires extra result=killed parameter in querystring
	if status == "canceled" {
		url += "&result=killed"
	}
	if G_IsWindows {
		if langutil.GetDefaultLang() != 0x409 {
			tmp, _ := langutil.GbkToUtf8([]byte(output))
			output = string(tmp)
		}
	}
	var err error
	_, err = util.HttpPost(url, output, "text")

	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		_, err = util.HttpPost(url, output, "text")
	}

}

func (task *Task) SendError(output string, errCode presetWrapErrorCode, errDesc string) {
	safelyTruncatedErrDesc := langutil.SafeTruncateStringInBytes(errDesc, 255)
	escapedErrDesc := url.QueryEscape(safelyTruncatedErrDesc)
	queryString := fmt.Sprintf("?taskId=%s&start=%d&end=%d&exitCode=%d&dropped=%d&errCode=%d&errDesc=%s",
		task.taskInfo.TaskId, task.startTime, task.endTime, task.exit_code,
		task.droped, errCode, escapedErrDesc)
	requestURL := util.GetErrorOutputService() + queryString

	if len(output) > 0 && G_IsWindows {
		if langutil.GetDefaultLang() != 0x409 {
			tmp, _ := langutil.GbkToUtf8([]byte(output))
			output = string(tmp)
		}
	}

	_, err := util.HttpPost(requestURL, output, "text")
	for i := 0; i < 3 && err != nil; i++ {
		time.Sleep(time.Duration(2) * time.Second)
		_, err = util.HttpPost(requestURL, output, "text")
	}
}

func (task *Task) Cancel() {
	task.cancelMut.Lock()
	defer task.cancelMut.Unlock()
	task.canceled = true
	// Consistent with C++ version, end time of canceled task is set to the time
	// of cancel operation
	task.endTime = timetool.GetAccurateTime()
	task.sendOutput("canceled", task.getReportString(task.output))
	task.processer.Cancel()
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

func (task *Task) sendRunningOutput(data string) {
	url := util.GetRunningOutputService()
	url += "?taskId=" + task.taskInfo.TaskId + "&start=" + strconv.FormatInt(task.startTime, 10)
	if len(data) == 0 && task.taskInfo.Output.SkipEmpty {
		return
	}
	if G_IsWindows {
		if langutil.GetDefaultLang() != 0x409 {
			tmp, _ := langutil.GbkToUtf8([]byte(data))
			data = string(tmp)
		}
	}
	util.HttpPost(url, data, "text")
}

func (task *Task) IsCancled() bool {
	task.cancelMut.Lock()
	defer task.cancelMut.Unlock()
	return task.canceled
}
