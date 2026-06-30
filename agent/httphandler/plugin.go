package httphandler

import (
	"encoding/json"
	"net/http"

	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	rpc "github.com/aliyun/aliyun_assist_client/interprocess/httpserver/jsonrpc"
)

const (
	plugin_reportEvent_method = "plugin.reportEvent"
)

type pluginReportEventParam struct {
	Plugin  string `json:"plugin"`
	Version string `json:"version"`
	EventID string `json:"eventID"`
	Content string `json:"content"`
}

func PluginRPCHandler(w http.ResponseWriter, r *http.Request) {
	rpc.ProcessRPCRequestFromHTTP(w, r, pluginRPC)
}

func pluginRPC(req *rpc.RPCRequest) (resp *rpc.RPCResponse) {
	switch req.Method {
	case plugin_reportEvent_method:
		// No need response for notification request
		if req.ID == nil {
			return nil
		}
		var param pluginReportEventParam
		if err := json.Unmarshal(*req.Params, &param); err != nil {
			resp = rpc.NewRPCErrorResponse(req.ID, rpc.ParseError(err))
			return
		}
		if param.Plugin == "" {
			resp = rpc.NewRPCErrorResponse(req.ID, rpc.InvalidParamsError("plugin"))
		} else if param.Version == "" {
			resp = rpc.NewRPCErrorResponse(req.ID, rpc.InvalidParamsError("version"))
		} else if param.EventID == "" {
			resp = rpc.NewRPCErrorResponse(req.ID, rpc.InvalidParamsError("eventID"))
		} else if param.Content == "" {
			resp = rpc.NewRPCErrorResponse(req.ID, rpc.InvalidParamsError("content"))
		}
		if resp != nil {
			return
		}

		metrics.GetPluginCustomizedEvent(
			"pluginName", param.Plugin,
			"pluginVersion", param.Version,
			"eventId", param.EventID,
			"content", param.Content,
		).ReportEvent()
		resp = rpc.NewRPCResultResponse(req.ID, "ok")
	default:
		resp = rpc.NewRPCErrorResponse(req.ID, rpc.MethodNotFoundError(req.Method))
	}
	return
}
