package jsonrpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wI2L/jsondiff"
)

type mockHTTPResponseWriter struct {
	statusCode int
	header     http.Header
	body       []byte
}

func (m *mockHTTPResponseWriter) Header() http.Header {
	return m.header
}
func (m *mockHTTPResponseWriter) Write(b []byte) (int, error) {
	m.body = b
	return len(b), nil
}
func (m *mockHTTPResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
	return
}

type readCloser struct {
	bytes.Buffer
}

func (r readCloser) Close() error {
	return nil
}

// Test cases reference https://www.jsonrpc.org/specification
func TestJsonRPC(t *testing.T) {
	type expectReqResp struct {
		req  string
		resp string
	}

	testCases := []struct {
		name    string
		reqresp []expectReqResp
	}{
		{
			name: "rpc call with positional parameters",
			reqresp: []expectReqResp{
				{
					req:  `{"jsonrpc": "2.0", "method": "subtract", "params": [42, 23], "id": 1}`,
					resp: `{"jsonrpc": "2.0", "result": 19, "id": 1}`,
				},
				{
					req:  `{"jsonrpc": "2.0", "method": "subtract", "params": [23, 42], "id": 2}`,
					resp: `{"jsonrpc": "2.0", "result": -19, "id": 2}`,
				},
			},
		},
		{
			name: "rpc call with named parameters",
			reqresp: []expectReqResp{
				{
					req:  `{"jsonrpc": "2.0", "method": "subtract_named", "params": {"subtrahend": 23, "minuend": 42}, "id": 3}`,
					resp: `{"jsonrpc": "2.0", "result": 19, "id": 3}`,
				},
				{
					req:  `{"jsonrpc": "2.0", "method": "subtract_named", "params": {"minuend": 42, "subtrahend": 23}, "id": 4}`,
					resp: `{"jsonrpc": "2.0", "result": 19, "id": 4}`,
				},
			},
		},
		{
			name: "a Notification",
			reqresp: []expectReqResp{
				{
					req:  `{"jsonrpc": "2.0", "method": "update_notification", "params": [1,2,3,4,5]}`,
					resp: ``,
				},
				{
					req:  `{"jsonrpc": "2.0", "method": "foobar_notification"}`,
					resp: ``,
				},
			},
		},
		{
			name: "rpc call of non-existent method",
			reqresp: []expectReqResp{
				{
					req:  `{"jsonrpc": "2.0", "method": "foobar", "id": "1"}`,
					resp: `{"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "1"}`,
				},
			},
		},
		{
			name: "rpc call with invalid JSON",
			reqresp: []expectReqResp{
				{
					req:  `{"jsonrpc": "2.0", "method": "foobar, "params": "bar", "baz]`,
					resp: `{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`,
				},
			},
		},
		{
			name: "rpc call with invalid Request object",
			reqresp: []expectReqResp{
				{
					req:  `{"jsonrpc": "2.0", "method": 1, "params": "bar"}`,
					resp: `{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
				},
			},
		},
		{
			name: "rpc call Batch, invalid JSON",
			reqresp: []expectReqResp{
				{
					req: `[
  {"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
  {"jsonrpc": "2.0", "method"
]`,
					resp: `{"jsonrpc": "2.0", "error": {"code": -32700, "message": "Parse error"}, "id": null}`,
				},
			},
		},
		{
			name: "rpc call with an empty Array",
			reqresp: []expectReqResp{
				{
					req:  `[]`,
					resp: `{"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}`,
				},
			},
		},
		{
			name: "rpc call with an invalid Batch (but not empty)",
			reqresp: []expectReqResp{
				{
					req: `[1]`,
					resp: `[
  {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
]`,
				},
			},
		},
		{
			name: "rpc call with invalid Batch",
			reqresp: []expectReqResp{
				{
					req: `[1,2,3]`,
					resp: `[
  {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
  {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
  {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null}
]`,
				},
			},
		},
		{
			name: "rpc call Batch",
			reqresp: []expectReqResp{
				{
					req: `[
        {"jsonrpc": "2.0", "method": "sum", "params": [1,2,4], "id": "1"},
        {"jsonrpc": "2.0", "method": "notify_hello", "params": [7]},
        {"jsonrpc": "2.0", "method": "subtract", "params": [42,23], "id": "2"},
        {"foo": "boo"},
        {"jsonrpc": "2.0", "method": "foo.get", "params": {"name": "myself"}, "id": "5"},
        {"jsonrpc": "2.0", "method": "get_data", "id": "9"} 
    ]`,
					resp: `[
        {"jsonrpc": "2.0", "result": 7, "id": "1"},
        {"jsonrpc": "2.0", "result": 19, "id": "2"},
        {"jsonrpc": "2.0", "error": {"code": -32600, "message": "Invalid Request"}, "id": null},
        {"jsonrpc": "2.0", "error": {"code": -32601, "message": "Method not found"}, "id": "5"},
        {"jsonrpc": "2.0", "result": ["hello", 5], "id": "9"}
    ]`,
				},
			},
		},
		{
			name: "rpc call Batch (all notifications)",
			reqresp: []expectReqResp{
				{
					req: `[
        {"jsonrpc": "2.0", "method": "notify_sum", "params": [1,2,4]},
        {"jsonrpc": "2.0", "method": "notify_hello", "params": [7]}
    ]`,
					resp: ``,
				},
			},
		},
	}

	for _, tc := range testCases {
		for idx, reqresp := range tc.reqresp {
			t.Run(fmt.Sprintf("%s: %d", tc.name, idx), func(t *testing.T) {
				buf := bytes.NewBuffer([]byte(reqresp.req))
				rc := readCloser{*buf}
				w := &mockHTTPResponseWriter{
					header: make(http.Header),
					body:   make([]byte, 0),
				}
				r := &http.Request{
					Header: make(http.Header),
					Body:   &rc,
				}

				ProcessRPCRequestFromHTTP(w, r, testRPC)

				fmt.Println("========================", t.Name())
				fmt.Println("resp: ", string(w.body))
				fmt.Println("expected resp: ", string(reqresp.resp))
				// fmt.Printf("[%s], [%s]\n", string(w.body), reqresp.resp)
				if string(w.body) != reqresp.resp {
					patch, err := jsondiff.CompareJSON(w.body, []byte(reqresp.resp))
					if len(patch) > 0 {
						// ignore path:/error/data
						pp := []jsondiff.Operation{}
						for idx, p := range patch {
							if !strings.HasSuffix(p.Path, "/error/data") {
								pp = append(pp, patch[idx])
							}
						}
						patch = pp
					}
					fmt.Println("json diff: ", patch, err)
					assert.Zero(t, len(patch))
					assert.Nil(t, err)
				}
			})
		}
	}
}

func testRPC(req *RPCRequest) (resp *RPCResponse) {
	switch req.Method {
	case "subtract":
		// No need response for notification request
		if req.ID == nil {
			return nil
		}
		var param []int
		if err := json.Unmarshal(*req.Params, &param); err != nil {
			resp = NewRPCErrorResponse(req.ID, ParseError(err))
			return
		}
		res := param[0] - param[1]
		resp = NewRPCResultResponse(req.ID, res)
	case "subtract_named":
		// No need response for notification request
		if req.ID == nil {
			return nil
		}
		var param struct {
			Subtrahend int `json:"subtrahend"`
			Minuend    int `json:"minuend"`
		}
		if err := json.Unmarshal(*req.Params, &param); err != nil {
			resp = NewRPCErrorResponse(req.ID, ParseError(err))
			return
		}
		res := param.Minuend - param.Subtrahend
		resp = NewRPCResultResponse(req.ID, res)
	case "sum":
		// No need response for notification request
		if req.ID == nil {
			return nil
		}
		var param []int
		if err := json.Unmarshal(*req.Params, &param); err != nil {
			resp = NewRPCErrorResponse(req.ID, ParseError(err))
			return
		}
		res := 0
		for _, p := range param {
			res += p
		}
		resp = NewRPCResultResponse(req.ID, res)
	case "get_data":
		// No need response for notification request
		if req.ID == nil {
			return nil
		}
		res := []any{"hello", 5}
		resp = NewRPCResultResponse(req.ID, res)
	case "update_notification", "foobar_notification", "notify_hello":
		resp = nil
	default:
		resp = NewRPCErrorResponse(req.ID, MethodNotFoundError(req.Method))
	}
	return
}
