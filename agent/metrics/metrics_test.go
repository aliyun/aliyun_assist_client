package metrics

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/jarcoal/httpmock"
	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/internal/testutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func TestMetrics(t *testing.T) {
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
		fmt.Sprintf("https://%s.axt.aliyun.com/luban/api/metrics", mockRegionId),
		func(h *http.Request) (*http.Response, error) {
			return httpmock.NewStringResponse(200, "success"), nil
		})

	GetChannelFailEvent(
		EVENT_SUBCATEGORY_CHANNEL_GSHELL,
		"key", "value",
	).ReportEvent()
	
	GetChannelSwitchEvent(
		"key", "value",
	).ReportEvent()

	GetUpdateFailedEvent(
		"key", "value",
	).ReportEvent()

	GetTaskFailedEvent(
		"key", "value",
	).ReportEvent()

	GetHybridRegisterEvent(
	).ReportEvent()
	GetHybridRegisterEvent(
	).ReportEvent()

	GetHybridUnregisterEvent(
	).ReportEvent()
	GetHybridUnregisterEvent(
	).ReportEvent()

	GetSessionFailedEvent(
		"key", "value",
	).ReportEvent()

	GetCpuOverloadEvent(
		"key", "value",
	).ReportEvent()

	GetMemOverloadEvent(
		"key", "value",
	).ReportEvent()

	GetBaseStartupEvent(
		"key", "value",
	).ReportEvent()

	url := util.GetMetricsService()
	doReport(url, "content")
}
