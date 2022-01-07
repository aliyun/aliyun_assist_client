package metrics

import (
	"testing"
	"net/http"
	"fmt"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/jarcoal/httpmock"
)

func TestMetrics(t *testing.T) {
	httpmock.Activate()
	util.NilRequest.Set()
	defer httpmock.DeactivateAndReset()
	defer util.NilRequest.Clear()
	mockRegionId := "mock-region"
	util.MockMetaServer(mockRegionId)
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
		true,
		"key", "value",
	).ReportEvent()
	GetHybridRegisterEvent(
		false,
		"key", "value",
	).ReportEvent()

	GetHybridUnregisterEvent(
		true,
		"key", "value",
	).ReportEvent()
	GetHybridUnregisterEvent(
		false,
		"key", "value",
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
