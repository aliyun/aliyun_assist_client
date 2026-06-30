package channel

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/clientreport"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestGshellChannel(t *testing.T) {
	flagging.InitConfig(logrus.New())

	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()

	httpmock.Activate()
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
				Code:          100,
				GshellSupport: "true",
				InstanceID:    "gshell-id",
				RequestID:     "request-id",
				Retry:         2,
			}
			gshellstatus.ThrottlingConfig.MaxKickVmCount = 10
			gshellstatus.ThrottlingConfig.MaxKickVmPeriod = 10
			gshellstatus.ThrottlingConfig.WssCoolDownCount = 1
			gshellstatus.ThrottlingConfig.WssCoolDownTime = 60
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
	guard_gshellPath := gomonkey.ApplyFunc(getGshellPath, func() (string, error) {
		return mockGshellFile, nil
	})
	defer func() {
		guard_gshellPath.Reset()
	}()
	go func() {
		gshellResp := []byte("{\"execute\":\"guest-sync\"}")
		f, err := os.OpenFile(mockGshellFile, os.O_WRONLY|os.O_APPEND, os.ModePerm)
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
	path, _ := pathutil.GetHybridPath(pathutil.GPNoCreation)
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
	requester.NilTransport.Set()
	defer requester.NilTransport.Clear()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	defer gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	}).Reset()

	defer gomonkey.ApplyFunc(clientreport.SendReport, func(clientreport.ClientReport) (string, error) {
		return "", nil
	}).Reset()

	path, _ := pathutil.GetHybridPath()
	machine_path := filepath.Join(path, "machine-id")
	instance_path := filepath.Join(path, "instance-id")
	fileutil.WriteStringToFile(machine_path, "machine-id")
	fileutil.WriteStringToFile(instance_path, "instance-id")
	defer func() {
		os.Remove(machine_path)
		os.Remove(instance_path)
	}()
	defer gomonkey.ApplyFunc(uuid.New, func() uuid.UUID {
		uuid := [16]byte{}
		for i := range uuid {
			uuid[i] = defaultLetters[rand.Intn(len(defaultLetters))]
		}
		return uuid
	}).Reset()

	mockServer, mockWss, err := createMockServerWss(echo)
	assert.Nil(t, err)
	defer mockServer.Close()
	defer mockWss.Close()

	t.Run("DialFailed", func (t *testing.T) {
		var mockErr = errors.New("dial failed")
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return nil, nil, mockErr
		}).Reset()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		err := wschannel.StartChannel()
		assert.Equal(t, mockErr, err)
	})
	t.Run("CalmDownForManyFailures", func (t *testing.T) {
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return nil, nil, errors.New("dial failed")
		}).Reset()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		wschannel.consecutiveConnectFailed = 0
		for i := 0; i < 3; i += 1 {
			wschannel.StartChannel()
			t.Log("ws_channel_calming_down: ", wschannel.consecutiveConnectFailed)
			t.Log("ws_channel_calming_down: ", time.Now().Before(wschannel.calmDownUntil))
			t.Log("ws_channel_calming_down: ", time.Now().Format("2006-01-02 15:04:05"))
			t.Log("ws_channel_calming_down: ", wschannel.calmDownUntil.Format("2006-01-02 15:04:05"))
		}
		wschannel.consecutiveConnectFailed = wssCoolDownCount
		err := wschannel.StartChannel()
		assert.Equal(t, errors.New("ws channel is calming down"), err)
	})
	t.Run("DialFailedWithCertificateError", func (t *testing.T) {
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return nil, nil, &tls.CertificateVerificationError{}
		}).Reset()
		accumulateRootCAs := false
		defer gomonkey.ApplyFunc(requester.AccumulateRootCAs, func(logrus.FieldLogger) func(func(*x509.CertPool) bool) {
			accumulateRootCAs = true
			return func(func(*x509.CertPool) bool) {}
		}).Reset()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		wschannel.StartChannel()
		assert.True(t, accumulateRootCAs)
	})
	t.Run("DialFailedWithResponse", func (t *testing.T) {
		var mockResponse = &http.Response{
			StatusCode: 500,
			Body: io.NopCloser(bytes.NewBufferString("mock response")),
		}
		var mockErr = errors.New("mock dial error with response")
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return nil, mockResponse, mockErr
		}).Reset()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		err := wschannel.StartChannel()
		assert.ErrorIs(t, err, mockErr)
	})
	t.Run("ReadMessageFailed", func (t *testing.T) {
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return mockWss, nil, nil
		}).Reset()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		err := wschannel.StartChannel()
		assert.Nil(t, err)
		for wschannel.IsWorking() {
			time.Sleep(time.Second)
		}
	})
	t.Run("PingFailed", func (t *testing.T) {
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return mockWss, nil, nil
		}).Reset()
		wssPingIntervalBk := wssPingInterval
		wssPingInterval = time.Second * 1
		defer func() { wssPingInterval = wssPingIntervalBk }()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		err := wschannel.StartChannel()
		assert.Nil(t, err)
		for wschannel.IsWorking() {
			time.Sleep(time.Second)
		}
	})
	t.Run("ActiveClose", func (t *testing.T) {
		var c *websocket.Dialer
		defer gomonkey.ApplyMethod(reflect.TypeOf(c), "Dial", func(*websocket.Dialer, string, http.Header) (*websocket.Conn, *http.Response, error) {
			return mockWss, nil, nil
		}).Reset()

		wschannel := NewWebsocketChannel(OnRecvMsg)
		wschannel.IsSupported()
		err := wschannel.StartChannel()
		assert.Nil(t, err)
		wschannel.StopChannel()
	})
}

func createMockServerWss(handler http.HandlerFunc) (*httptest.Server, *websocket.Conn, error) {
	// Create test server with the echo handler.
	s := httptest.NewServer(http.HandlerFunc(echo))

	// Convert http://127.0.0.1 to ws://127.0.0.
	u := "ws" + strings.TrimPrefix(s.URL, "http")

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	return s, ws, err
}

var upgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	idx := 10
	for {
		idx += 1
		// send kick_vm
		err = c.WriteMessage(websocket.TextMessage, []byte("kick_vm"))
		if err != nil {
			break
		}
		if idx >= 3 {
			// connection break
			return
		}

		_, _, err := c.ReadMessage()
		if err != nil {
			break
		}
		time.Sleep(time.Second)
	}
}
