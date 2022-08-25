package taskengine

import (
	"encoding/base64"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"runtime"
	"strconv"
	"testing"
	"time"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
)

func addMockServer() {
	httpmock.Activate()

	httpmock.RegisterResponder("GET", "http://100.100.100.200/latest/meta-data/region-id",
		httpmock.NewStringResponder(200, `cn-test`))

	httpmock.RegisterResponder("GET", "https://cn-test.axt.aliyun.com/luban/api/connection_detect",
		httpmock.NewStringResponder(200, `ok`))

	httpmock.RegisterResponder("POST", "https://cn-test.axt.aliyun.com//luban/api/v1/task/finish",
		httpmock.NewStringResponder(200, ``))

	httpmock.RegisterResponder("POST", "https://cn-test.axt.aliyun.com//luban/api/v1/task/running",
		httpmock.NewStringResponder(200, ``))
}

func removeMockServer() {
	httpmock.DeactivateAndReset()
}

func TestRunTask(t *testing.T) {
	addMockServer()
	defer removeMockServer()

	var commandType string
	var content string
	var workingDir string
	if runtime.GOOS == "linux" {
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
		TaskId:  "t-test" + rand_str,
		CommandId: "c-test",
		TimeOut:"120",
		WorkingDir:workingDir,
		Content:content,
	}
	task := NewTask(info, nil, nil)

	errcode, err := task.Run()

	/*if runtime.GOOS == "windows" {
		assert.Contains(t, output, "Users")
	}

	if runtime.GOOS == "linux" {
		assert.Contains(t, output, "tmp")
	}*/

	assert.Equal(t, nil , err)
	assert.Equal(t, 0 , int(errcode))
}