package pluginmanager

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
)

func TestPluginManager(t *testing.T) {
	guard := monkey.Patch(syncRunKillGroup, func(workingDir string, commandName string, commandArguments []string, stdoutWriter io.Writer, stderrWriter io.Writer,
		timeOut int) (exitCode int, status int, err error) {
			if commandName=="acs-plugin-manager" {
				if len(commandArguments)==1 && commandArguments[0]=="--status" {
					content := "[{\"arch\": \"x64\", \"name\": \"test_plugin_linux\", \"os\": \"linux\", \"pluginId\": \"local_test_plugin_linux_1.0\", \"status\": \"PERSIST_FAIL\", \"version\": \"1.0\"}]"
					stdoutWriter.Write([]byte(content))
					return 0, 0, nil
				}
			}
			return 1, 1, errors.New("unknown command")
		})
	defer guard.Unpatch()

	pluginPath, _ := pathutil.GetPluginPath()
	pluginPath += string(os.PathSeparator) + "installed_plugins"
	installed_plugins := "{\"pluginList\": [{\"arch\": \"x64\", \"isPreInstalled\": \"\", \"md5\": \"b085b7d7c0b88e27bbd8a0de8dd5caa2\", \"name\": \"test_plugin_linux\", \"osType\": \"linux\", \"pluginId\": \"local_test_plugin_linux_1.0\", \"pluginType\": 1, \"publisher\": \"aliyun\", \"runPath\": \"main\", \"timeout\": \"5\", \"url\": \"local\", \"version\": \"1.0\"}, {\"arch\": \"X64\", \"isPreInstalled\": \"\", \"md5\": \"b3696ef2e0add78c8f4601d598ba1daa\", \"name\": \"oosutil\", \"osType\": \"LINUX\", \"pluginId\": \"p-hz0100z6doel4hs\", \"pluginType\": 0, \"publisher\": \"aliyun\", \"runPath\": \"oosutil_linux\", \"timeout\": \"60\", \"url\": \"http://aliyun-client-assist-cn-hangzhou.oss-cn-hangzhou-internal.aliyuncs.com/oosutil/linux/oosutil_1.8.zip\", \"version\": \"1.8\"}]}"
	err := util.WriteStringToFile(pluginPath, installed_plugins)
	assert.Equal(t, err, nil)
	defer func() {
		if util.CheckFileIsExist(pluginPath) {
			os.Remove(pluginPath)
		}
	}()

	healthCheck := true
	updateCheck := true
	httpmock.Activate()
	util.NilRequest.Set()
	defer httpmock.DeactivateAndReset()
	defer util.NilRequest.Clear()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/plugin/status", mockRegion),
		func(h *http.Request) ( hp *http.Response, err error) {
			hp = httpmock.NewStringResponse(200, "success")
			err = nil
			content, _ := ioutil.ReadAll(h.Body)
			pluginStatusResq := PluginStatusResquest{}
			json.Unmarshal(content, &pluginStatusResq)
			if len(pluginStatusResq.Plugin) != 2 {
				healthCheck = false
				return
			}
			return 
		})
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/plugin/update_check", mockRegion),
		func(h *http.Request) ( hp *http.Response, err error) {
			err = nil
			content, _ := ioutil.ReadAll(h.Body)
			pluginUpdateCheckResq := PluginUpdateCheckRequest{}
			json.Unmarshal(content, &pluginUpdateCheckResq)
			pluginUpdateInfo := PluginUpdateInfo{}
			pluginUpdateInfo.Name = "name"
			pluginUpdateInfo.Version = "1.0"
			pluginUpdateCheckResp := PluginUpdateCheckResponse{
				InstanceId: "instance-id",
				NextInterval: 3000,
				Plugin: []PluginUpdateInfo{ pluginUpdateInfo },
			}
			resp_content, _ := json.Marshal(&pluginUpdateCheckResp)
			hp = httpmock.NewStringResponse(200, string(resp_content))
			if len(pluginUpdateCheckResq.Plugin) != 1 {
				updateCheck = false
				return
			}
			return 
		})
	setInterval("pluginHealthScanInterval", 5)
	setInterval("pluginUpdateCheckIntervalSeconds", 5)
	setInterval("unknows", 30)
	pluginHealthScanInterval = 5
	pluginUpdateCheckInterval = 5
	timermanager.InitTimerManager()
	InitPluginCheckTimer()
	time.Sleep(time.Duration(10) * time.Second)
	assert.Equal(t, healthCheck, true)
	assert.Equal(t, updateCheck, true)
}