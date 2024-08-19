package channel

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/google/uuid"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestGshellChannel(t *testing.T) {

	guard_transport := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard_transport.Reset()

	httpmock.Activate()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	defer httpmock.DeactivateAndReset()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/metrics", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/gshell", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			gshellstatus := gshellStatus{
				Code: 100,
				GshellSupport: "true",
				InstanceID: "gshell-id",
				RequestID: "request-id",
				Retry: 2,
			}
			resp, err := json.Marshal(&gshellstatus)
			if err != nil {
				return httpmock.NewStringResponse(502, "fail"), nil
			}
			return httpmock.NewStringResponse(200, string(resp)), nil
		})
	httpmock.RegisterResponder("POST",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/v1/exception/client_report", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	// mock gshell file
	var mockGshellFile string
	mockGshellFile = filepath.Join(os.TempDir(), "mockgshell")
	os.Create(mockGshellFile)
	defer os.Remove(mockGshellFile)
	guard_gshellPath := gomonkey.ApplyFunc(getGshellPath, func()(string, error) {
		return mockGshellFile, nil
	})
	defer func() {
		guard_gshellPath.Reset()
	}()
	go func() {
		gshellResp := []byte("{\"execute\":\"guest-sync\"}")
		f, err := os.OpenFile(mockGshellFile, os.O_WRONLY | os.O_APPEND, os.ModePerm)
		if err != nil {
			fmt.Println("open mock gshell file err: ", err)
			panic(0)
		}
		for {
			if _, err := f.Write(gshellResp); err != nil {
				fmt.Println("write mock gshell file err: ", err)
				return
			}
			time.Sleep(time.Second)
		}
	}()

	// mock instance-id file
	path,_ := pathutil.GetHybridPath()
	path = filepath.Join(path, "instance-id")
	if fileutil.CheckFileIsExist(path) {
		os.Remove(path)
	}

	var err error
	TryStartGshellChannel()
	err = InitChannelMgr(OnRecvMsg)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, 0, len(G_ChannelMgr.AllChannel))
	time.Sleep(time.Duration(2) * time.Second)

	_gshellChannel.StartChannel()
	if gshell, ok := _gshellChannel.(*GshellChannel); ok {
		gshell.WaitCheckDone.Done()
		gshell.SwitchChannel()
	}
	_gshellChannel.StopChannel()
}

var defaultLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func TestWSChannel(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)
	
	guard_host := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard_host.Reset()

	guard_clientReport := gomonkey.ApplyFunc(clientreport.SendReport, func(clientreport.ClientReport) (string, error) {
		return "", nil
	})
	defer guard_clientReport.Reset()


	wschannel := NewWebsocketChannel(OnRecvMsg)
	wschannel.IsSupported()

	path, _ := pathutil.GetHybridPath()
	machine_path := filepath.Join(path, "machine-id")
	instance_path := filepath.Join(path, "instance-id")
	fileutil.WriteStringToFile(machine_path, "machine-id")
	fileutil.WriteStringToFile(instance_path, "instance-id")
	defer func() {
		os.Remove(machine_path)
		os.Remove(instance_path)
	}()
	guard := gomonkey.ApplyFunc(uuid.New, func() uuid.UUID {
		uuid := [16]byte{}
		for i := range uuid {
			uuid[i] = defaultLetters[rand.Intn(len(defaultLetters))]
		}
		return uuid
	})
	wschannel.StartChannel()
	wschannel.SwitchChannel()
	wschannel.StartPings(time.Duration(100) * time.Millisecond)
	time.Sleep(time.Duration(500) * time.Microsecond)
	wschannel.StopChannel()
	guard.Reset()
}
