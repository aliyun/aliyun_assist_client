package taskengine

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/export"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/host"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskoutput"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskoutput/outputbuffer"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func addMockServer() {
	httpmock.Activate()

	httpmock.RegisterResponder("GET", "http://100.100.100.200/latest/meta-data/region-id",
		httpmock.NewStringResponder(200, `cn-test`))

	httpmock.RegisterResponder("GET", "https://cn-test.axt.aliyun.com/luban/api/connection_detect",
		httpmock.NewStringResponder(200, `ok`))

	httpmock.RegisterResponder("POST", "https://cn-test.axt.aliyun.com/luban/api/v1/task/finish",
		httpmock.NewStringResponder(200, `ok`))

	httpmock.RegisterResponder("POST", "https://cn-test.axt.aliyun.com/luban/api/v1/task/running",
		httpmock.NewStringResponder(200, `ok`))
}

func removeMockServer() {
	httpmock.DeactivateAndReset()
}

func TestRunTask(t *testing.T) {
	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()
	addMockServer()
	defer removeMockServer()
	guard := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return "cn-test.axt.aliyun.com"
	})
	defer guard.Reset()

	flagging.InitConfig(logrus.New())
	var commandType string
	var content string
	var workingDir string
	if osutil.GetOsType() == osutil.OSLinux || osutil.GetOsType() == osutil.OSFreebsd {
		commandType = "RunShellScript"
		content = base64.StdEncoding.EncodeToString([]byte("pwd"))
		workingDir = "/tmp"
	} else if runtime.GOOS == "windows" {
		commandType = "RunBatScript"
		content = base64.StdEncoding.EncodeToString([]byte("chdir"))
		workingDir = "C:\\Users"
	}

	rand.Seed(time.Now().UnixNano())
	rand_num := rand.Intn(10000000)
	rand_str := strconv.Itoa(rand_num)

	info := models.RunTaskInfo{
		InstanceId:  "i-test",
		CommandType: commandType,
		TaskId:      "t-test" + rand_str,
		CommandId:   "c-test",
		TimeOut:     "120",
		WorkingDir:  workingDir,
		Content:     content,
	}
	task, _ := NewTask(info, nil, nil, nil)

	errcode, err := task.Run()

	/*if runtime.GOOS == "windows" {
		assert.Contains(t, output, "Users")
	}

	if runtime.GOOS == "linux" {
		assert.Contains(t, output, "tmp")
	}*/

	assert.Equal(t, nil, err)
	assert.Equal(t, 0, int(errcode))
}

func TestLocalLauncher(t *testing.T) {
	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()
	addMockServer()
	defer removeMockServer()
	guard := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return "cn-test.axt.aliyun.com"
	})
	defer guard.Reset()
	flagging.InitConfig(logrus.New())

	var commandType string
	var content string
	var workingDir string
	rand.Seed(time.Now().UnixNano())
	rand_num := rand.Intn(10000000)
	rand_str := strconv.Itoa(rand_num)
	if osutil.GetOsType() == osutil.OSLinux || osutil.GetOsType() == osutil.OSFreebsd {
		commandType = "RunShellScript"
	} else if runtime.GOOS == "windows" {
		commandType = "RunBatScript"
	}
	t.Run("NormalLocalLauncher", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3",
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, err := task.Run()
		assert.Equal(t, nil, err)
		assert.Equal(t, 0, int(errcode))
	})
	t.Run("LocalLauncherNotExists", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "pyxx {{ACS::ScriptFileName}}",
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, _ := task.Run()
		assert.Equal(t, -23, int(errcode))
	})
	t.Run("LocalLauncherWithWrongParamCase1", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3 {{ADB::ScriptFileName}}", // Wrong prefix
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, _ := task.Run()
		assert.Equal(t, -12, int(errcode))
	})
	t.Run("LocalLauncherWithWrongParamCase2", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3 {{ACS::InstanceID}}", // Do not support other param
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, _ := task.Run()
		assert.Equal(t, -12, int(errcode))
	})
	t.Run("LocalLauncherWithWrongParamCase3", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3 {{ACSSCRIPT}}", // Ok whatever
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, _ := task.Run()
		assert.Equal(t, -12, int(errcode))
	})
	t.Run("LocalLauncherWithWrongParamCase4", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3 {{ACS::ScriptFileName|Exx(.py)}}", // Wrong extension format
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, _ := task.Run()
		assert.Equal(t, -12, int(errcode))
	})
	t.Run("LocalLauncherWithWrongParamCase5", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3 {{ACS::ScriptFileName|Exx()}}", // Empty extension
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, _ := task.Run()
		assert.Equal(t, -12, int(errcode))
	})
	t.Run("ErrorContent", func(t *testing.T) {
		content = base64.StdEncoding.EncodeToString([]byte("print(1"))
		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: commandType,
			TaskId:      "t-test" + rand_str,
			CommandId:   "c-test",
			TimeOut:     "120",
			WorkingDir:  workingDir,
			Content:     content,
			Launcher:    "python3 {{ACS::ScriptFileName}}",
		}
		task, _ := NewTask(info, nil, nil, nil)
		errcode, err := task.Run()
		assert.Equal(t, nil, err)
		assert.Equal(t, 0, int(errcode))
	})
}

func TestLauncher(t *testing.T) {
	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()
	addMockServer()
	defer removeMockServer()
	guard := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return "cn-test.axt.aliyun.com"
	})
	defer guard.Reset()
	flagging.InitConfig(logrus.New())

	var commandType string
	var content string
	var workingDir string
	if osutil.GetOsType() == osutil.OSLinux || osutil.GetOsType() == osutil.OSFreebsd {
		commandType = "RunShellScript"
		// content = base64.StdEncoding.EncodeToString([]byte("filename='/home/fatdragonsxh/aliyun_assist_client/agent/taskengine/new_empty_file.txt'\nopen(filename,'w').close()\nprint(f\"'{filename}' created.\")"))
		content = base64.StdEncoding.EncodeToString([]byte("print(1)"))
		workingDir = "/tmp"
	} else if runtime.GOOS == "windows" {
		commandType = "RunBatScript"
		content = base64.StdEncoding.EncodeToString([]byte("chdir"))
		workingDir = "C:\\Users"
	}

	rand.Seed(time.Now().UnixNano())
	rand_num := rand.Intn(10000000)
	rand_str := strconv.Itoa(rand_num)

	info := models.RunTaskInfo{
		InstanceId:  "i-test",
		CommandType: commandType,
		TaskId:      "t-test" + rand_str,
		CommandId:   "c-test",
		TimeOut:     "120",
		WorkingDir:  workingDir,
		Content:     content,
		Launcher:    "python3 {{ACS::ScriptFileName|Ext(.py)}}",
	}
	task, _ := NewTask(info, nil, nil, nil)

	errcode, err := task.Run()

	/*if runtime.GOOS == "windows" {
		assert.Contains(t, output, "Users")
	}

	if runtime.GOOS == "linux" {
		assert.Contains(t, output, "tmp")
	}*/

	assert.Equal(t, nil, err)
	assert.Equal(t, 0, int(errcode))

}

func TestTaskCancel(t *testing.T) {
	var (
		taskStoppedResult  string
		taskStoppedErrCode string
		taskStoppedErrDesc string

		sendStoppedOutputCalled bool

		mockProcessorCancelErr error
	)

	guard_transport := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard_transport.Reset()

	httpmock.Activate()
	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	mockRegionId := "mock-region"
	testutil.MockMetaServer(mockRegionId)
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/task/stopped", mockRegionId),
		func(h *http.Request) (*http.Response, error) {
			sendStoppedOutputCalled = true
			taskStoppedResult = h.FormValue("result")
			taskStoppedErrCode = h.FormValue("errCode")
			taskStoppedErrDesc = h.FormValue("errDesc")
			return httpmock.NewStringResponse(200, "success"), nil
		})

	var p *host.HostProcessor
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), "Cancel", func(*host.HostProcessor) error {
		return mockProcessorCancelErr
	}).Reset()

	testCases := []struct {
		Name                        string
		Quietly                     bool
		IsTaskRunning               bool
		ExpectedStoppedOutputCalled bool
		CancelErrored               bool
	}{
		{
			Name:                        "quietly:false; taskrunning:false",
			Quietly:                     false,
			IsTaskRunning:               false,
			ExpectedStoppedOutputCalled: true,
		},
		{
			Name:                        "quietly:false; taskrunning:false; cancelError",
			Quietly:                     false,
			IsTaskRunning:               false,
			ExpectedStoppedOutputCalled: true,
			CancelErrored:               true,
		},
		{
			Name:                        "quietly:true; taskrunning:false",
			Quietly:                     true,
			IsTaskRunning:               false,
			ExpectedStoppedOutputCalled: false,
		},
		{
			Name:                        "quietly:false; taskrunning:true",
			Quietly:                     false,
			IsTaskRunning:               true,
			ExpectedStoppedOutputCalled: false,
		},
		{
			Name:                        "quietly:true; taskrunning:true",
			Quietly:                     true,
			IsTaskRunning:               true,
			ExpectedStoppedOutputCalled: false,
		},
	}

	for _, tc := range testCases {
		taskStoppedResult = ""
		taskStoppedErrCode = ""
		taskStoppedErrDesc = ""
		mockProcessorCancelErr = nil
		sendStoppedOutputCalled = false
		if tc.CancelErrored {
			mockProcessorCancelErr = errors.New("mock processor cancel error")
		}

		t.Run(tc.Name, func(t *testing.T) {
			info := models.RunTaskInfo{
				InstanceId:  "i-test",
				CommandType: "RunShellScript",
				TaskId:      "t-test",
				CommandId:   "c-test",
				TimeOut:     "120",
			}
			task, _ := NewTask(info, nil, nil, nil)
			task.canceled = false

			task.Cancel(tc.Quietly, tc.IsTaskRunning)

			assert.Equal(t, tc.ExpectedStoppedOutputCalled, sendStoppedOutputCalled)
			if sendStoppedOutputCalled {
				// task is not running when task.Cancel() called, so just call sendStoppedOutput() in
				// task.Cancel() with stopReasonKilled and no cancel-error.
				assert.Equal(t, stopReasonKilled, taskStoppedResult)
				assert.Equal(t, "", taskStoppedErrCode)
				assert.Equal(t, "", taskStoppedErrDesc)
			}
			if tc.IsTaskRunning {
				// task is running when task.Cancel() called, just set task.canceled flag and do not
				// call sendStoppedOutput() in task.Cancel(), sendStoppedOutput() will be called in task.Run().
				// Just set task.stopResult, task.cancelErr and task.needReportStopped in task.Cancel().
				assert.False(t, sendStoppedOutputCalled)
				assert.Equal(t, !tc.Quietly, task.needReportStopped)
				if tc.CancelErrored {
					assert.Equal(t, stopFailed, task.stopResult)
					assert.Equal(t, mockProcessorCancelErr, task.cancelErr)
				} else {
					assert.Equal(t, stopReasonKilled, task.stopResult)
					assert.Equal(t, nil, task.cancelErr)
				}
			}
		})

	}
}

func TestTaskRun_SendRunningOutput_MayFail(t *testing.T) {
	var (
		mockOutput     string
		expectedOutput string

		taskRunning     int
		taskRunningFlag bool
	)
	const (
		allOk = iota
		allFail
		FailOk
	)

	defer gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	}).Reset()

	flagging.InitConfig(logrus.New())
	httpmock.Activate()
	defer requester.NilTransport.Clear()
	defer httpmock.DeactivateAndReset()
	mockRegionId := "mock-region"
	defer gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return fmt.Sprintf("%s.axt.aliyun.com", mockRegionId)
	}).Reset()

	httpmock.RegisterResponder("POST", fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/task/finish", mockRegionId),
		func(r *http.Request) (*http.Response, error) {
			content, err := io.ReadAll(r.Body)
			assert.Nil(t, err)
			mockOutput += string(content)
			return httpmock.NewStringResponse(200, "success"), nil
		})
	httpmock.RegisterResponder("POST", fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/task/running", mockRegionId),
		func(r *http.Request) (*http.Response, error) {
			content, err := io.ReadAll(r.Body)
			assert.Nil(t, err)
			switch taskRunning {
			case allOk:
				mockOutput += string(content)
				return httpmock.NewStringResponse(200, "success"), nil
			case allFail:
				return httpmock.NewStringResponse(500, "error"), nil
			default:
				defer func() {
					taskRunningFlag = !taskRunningFlag
				}()
				if taskRunningFlag {
					mockOutput += string(content)
					return httpmock.NewStringResponse(200, "success"), nil
				}
				return httpmock.NewStringResponse(500, "error"), nil
			}
		})
	var ob *outputbuffer.OutputBuffer
	defer gomonkey.ApplyMethod(reflect.TypeOf(ob), "ReadPre", func() []byte {
		content := strconv.Itoa(rand.Intn(10))
		expectedOutput += content
		return []byte(content)
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(ob), "ReadAll", func() []byte {
		content := "finish"
		expectedOutput += content
		return []byte(content)
	}).Reset()
	var p *host.HostProcessor
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), "SyncRun", func(*host.HostProcessor, io.Writer, io.Writer, io.Reader) (int, int, error) {
		time.Sleep(time.Second * 5)
		return 0, process.Success, nil
	}).Reset()
	var tk *Task
	defer gomonkey.ApplyMethod(reflect.TypeOf(tk), "PreCheck", func(*Task, bool) error {
		return nil
	}).Reset()

	testCases := []struct {
		Name string
		Mode int
	}{
		{
			Name: "normal",
			Mode: allOk,
		},
		{
			Name: "task/running partial failure",
			Mode: FailOk,
		},
		{
			Name: "task/running all failure",
			Mode: allFail,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			mockOutput = ""
			expectedOutput = ""
			taskRunning = tt.Mode
			taskRunningFlag = false

			taskInfo := models.RunTaskInfo{
				TaskId:      "t-xxx",
				CommandType: "RunShellScript",
				Content:     "ZWNobyBoZWxsbw==",
			}
			task, err := NewTask(taskInfo, nil, nil, nil)
			assert.Nil(t, err)
			task.Run()
			assert.NotZero(t, len(mockOutput))
			assert.Equal(t, expectedOutput, mockOutput)
		})
	}
}

func TestParseReportResp_onReportError(t *testing.T) {
	testCases := []struct {
		Name           string
		TaskState      taskReportState
		TaskReportResp TaskReportResp
		OnReportError  bool
	}{
		{
			Name:      "task/finish; responseErr; onReportError",
			TaskState: TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{
				ErrorCode: "SomeErrorCode",
			},
			OnReportError: true,
		},
		{
			Name:      "task/finish; responseErr; onReportError:nil",
			TaskState: TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{
				ErrorCode: "SomeErrorCode",
			},
			OnReportError: false,
		},
		{
			Name:      "task/finish",
			TaskState: TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{
				ErrorCode: "",
			},
			OnReportError: true,
		},
	}

	for _, tc := range testCases {
		taskInfo := models.RunTaskInfo{
			TaskId:      "t-xxx",
			CommandType: "RunShellScript",
			Content:     "ZWNobyBoZWxsbw==",
		}
		task, err := NewTask(taskInfo, nil, nil, nil)
		assert.Nil(t, err)

		task.onReportError = func(taskId string, repeat models.RunTaskRepeatType, errorCode, status string) (isTaskErr bool) {
			return tc.OnReportError
		}

		content, err := json.Marshal(tc.TaskReportResp)
		assert.Nil(t, err)
		taskError := task.parseReportResp(tc.TaskState, string(content))
		if tc.TaskReportResp.ErrorCode != "" {
			if tc.OnReportError {
				assert.Equal(t, taskerrors.WrapErrServerResponseError, taskError.ErrCode())
			} else {
				assert.Nil(t, taskError)
			}
		} else {
			assert.Nil(t, taskError)
		}
	}
}

func TestParseReportResp_exportOutput(t *testing.T) {
	testCases := []struct {
		Name           string
		TaskState      taskReportState
		TaskReportResp TaskReportResp
		NeedExport     bool
		IsBufComplete  bool

		TaskStateShouldNotExportOss bool
	}{
		{
			Name:           "task/finish; NeedExport False",
			TaskState:      TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{},
			NeedExport:     false,
		},
		{
			Name:           "task/finish; no OssExporter",
			TaskState:      TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{},
			NeedExport:     true,
		},
		{
			Name:           "task/finish; buffer is Complete",
			TaskState:      TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  true,
		},
		{
			Name:           "task/finish; buffer is not Complete",
			TaskState:      TASK_STATE_FINISH,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
		},
		{
			Name:           "task/timeout; buffer is not Complete",
			TaskState:      TASK_STATE_TIMEOUT,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
		},
		{
			Name:           "task/stopped; buffer is not Complete",
			TaskState:      TASK_STATE_STOPPED,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
		},
		{
			Name:           "task/error; buffer is not Complete",
			TaskState:      TASK_STATE_ERROR,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
		},

		{
			Name:           "task/invalid; url should not export",
			TaskState:      TASK_STATE_INVALID,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
			TaskStateShouldNotExportOss: true,
		},
		{
			Name:           "task/running; url should not export",
			TaskState:      TASK_STATE_RUNNING,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
			TaskStateShouldNotExportOss: true,
		},
		{
			Name:           "task/verified; url should not export",
			TaskState:      TASK_STATE_VERIFIED,
			TaskReportResp: TaskReportResp{
				OssExporter: []*export.OSSExporter{
					{
						OssURL: "http://invalid",
					},
				},
			},
			NeedExport:     true,
			IsBufComplete:  false,
			TaskStateShouldNotExportOss: true,
		},
	}

	var (
		mockIsBufComplete bool
		
		mockBufferReadAllFromStart = []byte("hello")

		receivedBufferContent []byte
		outputReleaseCalled int
		exportToOssCalled int
	)
	var b *taskoutput.MultiWriter
	defer gomonkey.ApplyMethod(reflect.TypeOf(b), "IsBufComplete", func(*taskoutput.MultiWriter) bool {
		return mockIsBufComplete
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(b), "Release", func(*taskoutput.MultiWriter) {
		outputReleaseCalled += 1
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(b), "BufferReadAllFromStart", func(*taskoutput.MultiWriter) []byte {
		return mockBufferReadAllFromStart
	}).Reset()
	defer gomonkey.ApplyFunc(export.ExportToOss, func(logger logrus.FieldLogger, taskId string, invokeVersion int, content []byte, filePath string, fileCreateErr error, fileWriteErr error, confList []*export.OSSExporter){
		receivedBufferContent = content
		exportToOssCalled +=1
	}).Reset()

	for _, tc := range testCases {
		mockIsBufComplete = tc.IsBufComplete
		receivedBufferContent = nil
		outputReleaseCalled = 0
		exportToOssCalled = 0

		taskInfo := models.RunTaskInfo{
			TaskId:      "t-xxx",
			CommandType: "RunShellScript",
			Content:     "ZWNobyBoZWxsbw==",
			Output: models.OutputInfo{
				NeedExport: tc.NeedExport,
			},
		}
		task, err := NewTask(taskInfo, nil, nil, nil)
		assert.Nil(t, err)
		content, err := json.Marshal(tc.TaskReportResp)
		assert.Nil(t, err)
		task.output = taskoutput.NewMultiWriter(logrus.New(), nil)
		

		taskError := task.parseReportResp(tc.TaskState, string(content))
		assert.Nil(t, taskError)

		// wait export.ExportToOss() goroutine done
		time.Sleep(time.Second)

		if tc.TaskStateShouldNotExportOss {
			assert.Zero(t, outputReleaseCalled)
			assert.Zero(t, exportToOssCalled)
			return
		}

		if !tc.NeedExport {
			assert.Zero(t, outputReleaseCalled)
			assert.Zero(t, exportToOssCalled)
		} else {
			if len(tc.TaskReportResp.OssExporter) > 0 {
				// export.ExportToOss() be called
				assert.Equal(t, 1, exportToOssCalled)
				// output.Release() be called
				assert.Equal(t, 1, outputReleaseCalled)

				if tc.IsBufComplete {
					// export buffer content to oss
					assert.Equal(t, mockBufferReadAllFromStart, receivedBufferContent)
				}
			} else {
				// output.Release() be called
				assert.Equal(t, 1, outputReleaseCalled)
			}
		}
	}
}
