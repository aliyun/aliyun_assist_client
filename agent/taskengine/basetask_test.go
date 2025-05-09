package taskengine

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/host"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
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

	taskCanceledErr := func(t *testing.T) {
		taskStoppedResult = ""
		taskStoppedErrCode = ""
		taskStoppedErrDesc = ""
		mockProcessorCancelErr = errors.New("mock processor cancel error")

		sendStoppedOutputCalled = false

		info := models.RunTaskInfo{
			InstanceId:  "i-test",
			CommandType: "RunShellScript",
			TaskId:      "t-test",
			CommandId:   "c-test",
			TimeOut:     "120",
		}
		task, _ := NewTask(info, nil, nil, nil)
		task.canceled = false

		task.Cancel(false, false)
		assert.True(t, sendStoppedOutputCalled)
		assert.Equal(t, stopFailed, taskStoppedResult)
		assert.Equal(t, "TerminationException", taskStoppedErrCode)
		assert.Equal(t, mockProcessorCancelErr.Error(), taskStoppedErrDesc)
	}

	t.Run("taskCanceledErr", taskCanceledErr)
}
