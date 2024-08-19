package heartbeat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestBuildPingRequest(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	const virtType = "kvm"
	const osVersion = "Linux_#1 SMP Sun Jul 26 15:27:06 UTC 2020_x86_64"
	const uptime = 7136000
	const pid = 7136
	const processUptime = 409
	const acknowledgeCounter = 2
	const sendCounter = 2
	const azoneId = "1q23213"
	const isColdstart = false
	timestamp := time.Now().Unix()

	guard := gomonkey.ApplyFunc(util.GetServerHost, func() string {
		return mockRegion + ".axt.aliyun.com"
	})
	defer guard.Reset()
	// requestURL := buildPingRequest(virtType, osType, osVersion, appVersion, uptime,
	// 	timestamp, pid, processUptime, acknowledgeCounter, azoneId, isColdstart,
	// 	sendCounter)
	encodedOsVersion := url.QueryEscape(osVersion)
	querystring := fmt.Sprintf("?uptime=%d&timestamp=%d&pid=%d&process_uptime=%d&index=%d&seq_no=%d&virt_type=%s&os_version=%s&az=%s&machineid=%s&cold_start=%t",
		uptime, uint64(timestamp), pid, processUptime, acknowledgeCounter, sendCounter, virtType, encodedOsVersion, azoneId, _machineId, isColdstart)
	requestURL := "https://" + util.GetPingService() + querystring
	fmt.Println(requestURL)

	segments, err := url.Parse(requestURL)
	assert.NoErrorf(t, err, "buildPingRequest should not produce malformed URL: %s", requestURL)
	assert.Exactly(t, "https", segments.Scheme, "Recommended to use HTTPS protocol")
	assert.Exactly(t, "/luban/api/heart-beat", segments.Path)

	params, err := url.ParseQuery(segments.RawQuery)
	assert.NoErrorf(t, err, "buildPingRequest should not produce malformed querystring: %s", requestURL)

	// Special case for os_version param due to value escaping
	assert.Exactly(t, 1, len(params["os_version"]))
	unescapedOsTypeParam, err := url.QueryUnescape(params["os_version"][0])
	assert.NoErrorf(t, err, "buildPingRequest should not produce malformed os_version parameter in querystring: %s", requestURL)
	assert.Exactly(t, osVersion, unescapedOsTypeParam)

	// Ordinary case
	var paramCases = []struct {
		expected     string
		actualValues []string
	}{
		{virtType, params["virt_type"]},
		{strconv.Itoa(uptime), params["uptime"]},
		{fmt.Sprintf("%d", uint64(timestamp)), params["timestamp"]},
		{strconv.Itoa(pid), params["pid"]},
		{strconv.Itoa(processUptime), params["process_uptime"]},
		{strconv.Itoa(acknowledgeCounter), params["index"]},
		{strconv.Itoa(sendCounter), params["seq_no"]},
	}
	for _, c := range paramCases {
		assert.Exactly(t, 1, len(c.actualValues))
		assert.Exactly(t, c.expected, c.actualValues[0])
	}
}

func generateFakePingRequest(mockRegion string) string {
	const virtType = "kvm"
	const osType = "linux"
	const osVersion = "Linux_#1 SMP Sun Jul 26 15:27:06 UTC 2020_x86_64"
	const appVersion = "2.0.5.7136"
	const uptime = 7136000
	const timestamp = 1595777226000
	const pid = 7136
	const processUptime = 409
	const acknowledgeCounter = 2
	const sendCounter = 2

	var requestURL = url.URL{
		// Scheme: "https",
		Host: fmt.Sprintf("%s.axt.aliyun.com", mockRegion),
		Path: "/luban/api/heart-beat",
		RawQuery: url.Values{
			"virt_type":      []string{virtType},
			"os_type":        []string{osType},
			"os_version":     []string{url.QueryEscape(osVersion)},
			"app_version":    []string{appVersion},
			"uptime":         []string{strconv.Itoa(uptime)},
			"timestamp":      []string{fmt.Sprintf("%d", uint64(timestamp))},
			"pid":            []string{strconv.Itoa(pid)},
			"process_uptime": []string{strconv.Itoa(processUptime)},
			"index":          []string{strconv.Itoa(acknowledgeCounter)},
			"seq_no":         []string{strconv.Itoa(sendCounter)},
		}.Encode(),
	}
	return strings.TrimLeft(requestURL.String(), "/")
}

func generateFakeSuccessfulResponseOrPanic() string {
	responseBytes, err := json.Marshal(map[string]interface{}{
		"code":         200,
		"instanceId":   "i-localhost",
		"nextInterval": "1080",
		"newTasks":     false,
	})
	if err != nil {
		panic(err)
	}
	return string(responseBytes)
}

func generateFakeErrorResponseOrPanic() string {
	responseBytes, err := json.Marshal(map[string]interface{}{
		"code":       503,
		"instanceId": nil,
		"errCode":    nil,
		"errMsg":     "requestId: e7dc96f2-ac53-4901-8487-d3e9201e529d",
	})
	if err != nil {
		panic(err)
	}
	return string(responseBytes)
}

func TestInvokePingRequest(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	mockRequestURLWithoutScheme := generateFakePingRequest(mockRegion)
	mockResponse := generateFakeSuccessfulResponseOrPanic()

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/heart-beat", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			assert.Exactly(t, "https://"+mockRequestURLWithoutScheme, h.URL.String(), "Mock server should receive same request as generated")
			return httpmock.NewStringResponse(200, mockResponse), nil
		})

	response, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)
	assert.NoError(t, err, "invokePingRequest should not return error for this testcase")
	assert.Exactly(t, mockResponse, response, "invokePingRequest should return mockResponse without error")
}

func TestInvokePingRequestRetrying(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()

	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	mockRequestURLWithoutScheme := generateFakePingRequest(mockRegion)
	mockSuccessfulResponse := generateFakeSuccessfulResponseOrPanic()
	mockErrorResponse := generateFakeErrorResponseOrPanic()

	var callCount uint32 = 0
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/heart-beat", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			assert.Exactly(t, "https://"+mockRequestURLWithoutScheme, h.URL.String(), "Mock server should receive same request as generated")

			if atomic.AddUint32(&callCount, 1) < 2 {
				return httpmock.NewStringResponse(503, mockErrorResponse), nil
			}
			return httpmock.NewStringResponse(200, mockSuccessfulResponse), nil
		})

	response, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)

	assert.NoError(t, err, "invokePingRequest should not return error for this testcase")
	assert.Exactly(t, mockSuccessfulResponse, response, "invokePingRequest should return mockResponse without error")

}

func TestInvokePingRequestRetryingWithLimit(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	_retryCounter = 0
	defer func() {
		_retryCounter = 0
	}()

	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	mockRequestURLWithoutScheme := generateFakePingRequest(mockRegion)
	mockSuccessfulResponse := generateFakeSuccessfulResponseOrPanic()
	mockErrorResponse := generateFakeErrorResponseOrPanic()

	var callCount uint32 = 0
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/heart-beat", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			assert.Exactly(t, "https://"+mockRequestURLWithoutScheme, h.URL.String(), "Mock server should receive same request as generated")

			if atomic.AddUint32(&callCount, 1) < 2 {
				return httpmock.NewStringResponse(503, mockErrorResponse), nil
			}
			atomic.StoreUint32(&callCount, 0)
			return httpmock.NewStringResponse(200, mockSuccessfulResponse), nil
		})

	for i := 0; i < 3; i++ {
		response, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)

		assert.NoError(t, err, "invokePingRequest should not return error for this testcase")
		assert.Exactly(t, mockSuccessfulResponse, response, "invokePingRequest should return mockResponse without error")
	}

	_, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)
	assert.Error(t, err, "invokePingRequest should return error for this testcase")

}

func TestInvokePingRequestNetworkError(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()

	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	mockRequestURLWithoutScheme := generateFakePingRequest(mockRegion)

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/heart-beat", mockRegion),
		httpmock.NewStringResponder(503, "https://"+mockRequestURLWithoutScheme))

	response, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)

	assert.Error(t, err, "invokePingRequest should return error for this testcase")
	fmt.Println(response, err)
	assert.Empty(t, response, "invokePingRequest should hide response to caller when error encountered")
}

func TestInvokePingRequestServerError(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()

	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)

	mockRequestURLWithoutScheme := generateFakePingRequest(mockRegion)
	mockErrorResponse := generateFakeErrorResponseOrPanic()

	httpmock.RegisterResponder("GET",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/heart-beat", mockRegion),
		httpmock.NewStringResponder(503, mockErrorResponse))

	response, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)

	assert.Error(t, err, "invokePingRequest should return error for this testcase")
	assert.Empty(t, response, "invokePingRequest should hide response to caller when error encountered")
}

func TestInvokePingRequestTimeOut(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	const mockRegion = "cn-test500"
	testutil.MockMetaServer(mockRegion)

	mockRequestURLWithoutScheme := generateFakePingRequest(mockRegion)
	mockSuccessfulResponse := generateFakeSuccessfulResponseOrPanic()
	mockErrorResponse := generateFakeErrorResponseOrPanic()

	var callCount uint32 = 0
	httpmock.RegisterResponder("GET",
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/heart-beat", mockRegion),
		func(h *http.Request) (*http.Response, error) {
			assert.Exactly(t, "https://"+mockRequestURLWithoutScheme, h.URL.String(), "Mock server should receive same request as generated")
			if atomic.AddUint32(&callCount, 1) < 2 {
				time.Sleep(time.Second * 20)
				return httpmock.NewStringResponse(404, mockErrorResponse), nil
			}
			return httpmock.NewStringResponse(200, mockSuccessfulResponse), nil
		})

	response, err := invokePingRequest(false, mockRequestURLWithoutScheme, false)

	assert.NoError(t, err, "invokePingRequest should not return error for this testcase")
	assert.Exactly(t, mockSuccessfulResponse, response, "invokePingRequest should return mockResponse without error")
}

var (
	// Test protocol switch between http and https
	enableHttpx = func(scheme, mockRegion string) {
		httpmock.RegisterResponder("GET", fmt.Sprintf("%s://%s.axt.aliyun.com/luban/api/heart-beat", scheme, mockRegion),
			httpmock.NewStringResponder(200, "ok"))
	}
	disableHttpx = func(scheme, mockRegion string) {
		httpmock.RegisterResponder("GET", fmt.Sprintf("%s://%s.axt.aliyun.com/luban/api/heart-beat", scheme, mockRegion),
			httpmock.NewStringResponder(500, "not ok"))
	}
)

func Test_doPing(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	defer func() {
		_retryCounter = 0
	}()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)
	timermanager.InitTimerManager()
	InitHeartbeatTimer()
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "invokePingRequestError",
			wantErr: true,
		},
		{
			name:    "nextIntervalNotExist",
			wantErr: false,
		},
		{
			name:    "nextIntervalNotFloat64",
			wantErr: false,
		},
		{
			name:    "newTasksNotExist",
			wantErr: false,
		},
		{
			name:    "newTasksNotBool",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := make(map[string]interface{})
			res["nextInterval"] = 100
			res["newTasks"] = true
			if tt.name == "nextIntervalNotExist" {
				delete(res, "nextInterval")
			}
			if tt.name == "nextIntervalNotFloat64" {
				res["nextInterval"] = "abc"
			}
			if tt.name == "newTasksNotExist" {
				delete(res, "newTask")
			}
			if tt.name == "newTasksNotBool" {
				res["newTask"] = "abc"
			}

			content, _ := json.Marshal(&res)
			if tt.name == "invokePingRequestError" {
				guard := gomonkey.ApplyFunc(invokePingRequest, func(bool, string, bool) (string, error) { return "", errors.New("some error") })
				defer guard.Reset()
			} else {
				guard := gomonkey.ApplyFunc(invokePingRequest, func(bool, string, bool) (string, error) { return string(content), nil })
				defer guard.Reset()
			}
			if err := doPing(); (err != nil) != tt.wantErr {
				t.Errorf("doPing() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_doPingSwitchProtocol(t *testing.T) {

	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()

	// Mock instance.IsHybrid to make it must return the expect value
	// Else the return value is uncertain
	hybridGuard := gomonkey.ApplyFunc(instance.IsHybrid, func() bool { return false})
	
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	defer func() {
		_retryCounter = 0
	}()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)
	timermanager.InitTimerManager()
	InitHeartbeatTimer()
	var err error

	sleepGuard := gomonkey.ApplyFunc(time.Sleep, func(time.Duration) {})
	defer sleepGuard.Reset()

	disableHttpx("http", mockRegion)
	disableHttpx("https", mockRegion)
	_retryCounter = 0
	err = doPing()
	assert.EqualError(t, err, requester.NewHttpErrorCode(500).Error())

	// default protocol is http
	log.GetLogger().Info("Test: default protocol is http")
	_tryHttp = true
	enableHttpx("http", mockRegion)
	enableHttpx("https", mockRegion)
	_retryCounter = 0
	err = doPing()
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, _tryHttp, true)

	// if http not work, use https
	log.GetLogger().Info("Test: if http not work, use https")
	disableHttpx("http", mockRegion)
	enableHttpx("https", mockRegion)
	_retryCounter = 0
	err = doPing()
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, false, _tryHttp)

	// if switch protocol from http to https, keep use https until https not work
	log.GetLogger().Info("Test: if switch protocol from http to https, keep use https until https not work")
	enableHttpx("http", mockRegion)
	enableHttpx("https", mockRegion)
	_retryCounter = 0
	err = doPing()
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, _tryHttp, false)
	enableHttpx("http", mockRegion)
	disableHttpx("https", mockRegion)
	_retryCounter = 0
	err = doPing()
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, _tryHttp, true)

	// if http not work, use https and try http after 24 * 60 heart-beats
	log.GetLogger().Info("Test: if http not work, use https and try http after 24 * 60 heart-beats")
	disableHttpx("http", mockRegion)
	enableHttpx("https", mockRegion)
	_retryCounter = 0
	doPing()
	enableHttpx("http", mockRegion)
	for i := 0; i < 23*60; i += 1 {
		_retryCounter = 0
		err = doPing()
		_sendCounter++
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, false, _tryHttp)
	}
	for i := 0; i < 2*60; i += 1 {
		_retryCounter = 0
		err = doPing()
		_sendCounter++
		assert.ErrorIs(t, err, nil)
	}
	_retryCounter = 0
	err = doPing()
	_sendCounter++
	assert.ErrorIs(t, err, nil)
	assert.Equal(t, _tryHttp, true)

	_retryCounter = 0
	err = doPing()
	assert.ErrorIs(t, err, nil)

	hybridGuard.Reset()

	// use https and do not switch to http in hybrid mode
	log.GetLogger().Info("Test: use https and do not switch to http in hybrid mode")
	hybridGuard = gomonkey.ApplyFunc(instance.IsHybrid, func() bool { return true})
	defer hybridGuard.Reset()
	for i := 0; i < 25*60; i += 1 {
		disableHttpx("https", mockRegion)
		enableHttpx("http", mockRegion)
		_retryCounter = 0
		err = doPing()
		assert.EqualError(t, err, requester.NewHttpErrorCode(500).Error())
	}
	enableHttpx("https", mockRegion)
	_retryCounter = 0
	err = doPing()
	assert.ErrorIs(t, err, nil)
}

func TestRegisterActionWhenNetRecover(t *testing.T) {
	guard := gomonkey.ApplyFunc(requester.GetHTTPTransport, func(logrus.FieldLogger) *http.Transport {
		transport, _ := http.DefaultTransport.(*http.Transport)
		return transport
	})
	defer guard.Reset()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	util.NilRequest.Set()
	defer util.NilRequest.Clear()
	const mockRegion = "cn-test100"
	testutil.MockMetaServer(mockRegion)
	timermanager.InitTimerManager()
	InitHeartbeatTimer()

	enableNetwork := func() {
		enableHttpx("https", mockRegion)
		enableHttpx("http", mockRegion)
	}
	disableNetwork := func() {
		disableHttpx("https", mockRegion)
		disableHttpx("http", mockRegion)
	}

	_networkConnected.Store(false)
	flag := 0
	syncChan := make(chan int, 1)
	action := func() {
		flag += 1
		syncChan<-1
	}
	var err error
	RegisterActionWhenNetRecover(map[string]func(){
		"AddFlag": action,
	})
	enableNetwork()
	err = doPing()
	// wait for _actionsWhenNetRecover
	<-syncChan
	assert.Equal(t, nil, err)
	assert.Equal(t, 1, flag)
	for i:=1; i<5; i++ {
		disableNetwork()
		err = doPing()
		assert.NotEqual(t, nil, err)
		assert.Equal(t, i, flag)
		enableNetwork()
		err = doPing()
		// wait for _actionsWhenNetRecover
		<-syncChan
		assert.Equal(t, nil, err)
		assert.Equal(t, i+1, flag)
	}
}
