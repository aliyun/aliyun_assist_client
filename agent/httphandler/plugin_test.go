package httphandler

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	rpc "github.com/aliyun/aliyun_assist_client/interprocess/httpserver/jsonrpc"
	"github.com/stretchr/testify/assert"
)

func Test_PluginRPC(t *testing.T) {
	var (
		mockReqID       = json.RawMessage{'1'}
		metricsReported bool
	)
	var m *metrics.MetricsEvent
	defer gomonkey.ApplyMethod(reflect.TypeOf(m), "ReportEvent", func(*metrics.MetricsEvent) {
		metricsReported = true
	}).Reset()

	testCases := []struct {
		Name         string
		Req          *rpc.RPCRequest
		ReqParam     []byte
		ExpectedResp *rpc.RPCResponse
	}{
		{
			Name: "invalid_method",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "invalid_method",
				ID:      &mockReqID,
			},
			ExpectedResp: rpc.NewRPCErrorResponse(&mockReqID, rpc.MethodNotFoundError("invalid_method")),
		},
		{
			Name: "plugin.reportEvent notification",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      nil,
			},
			ExpectedResp: nil,
		},
		{
			Name: "plugin.reportEvent parseErr",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      &mockReqID,
			},
			ReqParam:     []byte("{"),
			ExpectedResp: rpc.NewRPCErrorResponse(&mockReqID, rpc.ParseError(errors.New("json decode err"))),
		},
		{
			Name: "plugin.reportEvent invalid param (missing plugin)",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      &mockReqID,
			},
			ReqParam: []byte(`{
				"version": "1.0",
				"eventID": "event-id",
				"content": "content"
			}`),
			ExpectedResp: rpc.NewRPCErrorResponse(&mockReqID, rpc.InvalidParamsError("plugin")),
		},
		{
			Name: "plugin.reportEvent invalid param (missing version)",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      &mockReqID,
			},
			ReqParam: []byte(`{
				"plugin": "p",
				"eventID": "event-id",
				"content": "content"
			}`),
			ExpectedResp: rpc.NewRPCErrorResponse(&mockReqID, rpc.InvalidParamsError("version")),
		},
		{
			Name: "plugin.reportEvent invalid param (missing eventID)",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      &mockReqID,
			},
			ReqParam: []byte(`{
				"plugin": "p",
				"version": "1.0",
				"content": "content"
			}`),
			ExpectedResp: rpc.NewRPCErrorResponse(&mockReqID, rpc.InvalidParamsError("eventID")),
		},
		{
			Name: "plugin.reportEvent invalid param (missing content)",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      &mockReqID,
			},
			ReqParam: []byte(`{
				"plugin": "p",
				"version": "1.0",
				"eventID": "event-id"
			}`),
			ExpectedResp: rpc.NewRPCErrorResponse(&mockReqID, rpc.InvalidParamsError("content")),
		},
		{
			Name: "plugin.reportEvent OK",
			Req: &rpc.RPCRequest{
				Version: "2.0",
				Method:  "plugin.reportEvent",
				ID:      &mockReqID,
			},
			ReqParam: []byte(`{
				"plugin": "p",
				"version": "1.0",
				"eventID": "event-id",
				"content": "content"
			}`),
			ExpectedResp: rpc.NewRPCResultResponse(&mockReqID, "ok"),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			metricsReported = false
			tt.Req.Params = (*json.RawMessage)(&tt.ReqParam)

			resp := pluginRPC(tt.Req)

			if resp != nil {
				assert.Equal(t, tt.ExpectedResp.Version, resp.Version)
				assert.Equal(t, tt.ExpectedResp.ID, resp.ID)
				if resp.Error != nil {
					assert.False(t, metricsReported)
					assert.Equal(t, tt.ExpectedResp.Error.Code, resp.Error.Code)
					assert.Equal(t, tt.ExpectedResp.Error.Message, resp.Error.Message)
				} else {
					assert.True(t, metricsReported)
					assert.Equal(t, "ok", resp.Result)
				}
			} else {
				assert.Nil(t, tt.Req.ID)
			}
		})
	}
}
