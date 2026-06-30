package jsonrpc

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type RPCRequest struct {
	Version string           `json:"jsonrpc"`
	Method  string           `json:"method"`
	ID      *json.RawMessage `json:"id"`
	Params  *json.RawMessage `json:"params,omitempty"`
}

type RPCResponse struct {
	Version string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *RPCError        `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

const (
	jsonRpcVersion = "2.0"
)

var Null = json.RawMessage([]byte("null"))

func NewRPCResultResponse(id *json.RawMessage, result interface{}) *RPCResponse {
	actualId := id
	if actualId == nil {
		actualId = &Null
	}
	return &RPCResponse{
		Version: jsonRpcVersion,
		ID:      actualId,
		Result:  result,
	}
}

func NewRPCErrorResponse(id *json.RawMessage, rpcErr *RPCError) *RPCResponse {
	actualId := id
	if actualId == nil {
		actualId = &Null
	}
	return &RPCResponse{
		Version: jsonRpcVersion,
		ID:      actualId,
		Error:   rpcErr,
	}
}

func (resp *RPCResponse) Marshal() ([]byte, error) {
	res, err := json.Marshal(resp)
	return res, err
}

type RPCHandler func(requests *RPCRequest) (responses *RPCResponse)

func ProcessRPCRequestFromHTTP(w http.ResponseWriter, r *http.Request, handler RPCHandler) {
	rawRequests, batch := decodeRawReqFromHTTP(w, r)
	if len(rawRequests) == 0 {
		return
	}

	// Process rpc request one by one
	var responses []*RPCResponse
	for _, raw := range rawRequests {
		request, response := decodeSingleReq(raw)
		if request != nil {
			response = handler(request)
			if request.ID != nil {
				// No need response for notification request
				responses = append(responses, response)
			}
		} else {
			responses = append(responses, response)
		}
	}

	encodeRespToHTTP(w, batch, responses...)
}

func decodeRawReqFromHTTP(w http.ResponseWriter, r *http.Request) (rawRequests []*json.RawMessage, batch bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		batch = false
		encodeRespToHTTP(w, batch, NewRPCErrorResponse(nil, ParseError(err)))
		return
	}
	if !json.Valid(body) {
		batch = false
		encodeRespToHTTP(w, batch, NewRPCErrorResponse(nil, ParseError(errors.New("Invalid JSON"))))
		return
	}
	if err := json.Unmarshal(body, &rawRequests); err != nil {
		batch = false
		rawRequests = append(rawRequests, (*json.RawMessage)(&body))
	} else if len(rawRequests) == 0 {
		batch = false
		encodeRespToHTTP(w, batch, NewRPCErrorResponse(nil, InvalidRequestError(errors.New("no request found"))))
		return
	} else {
		batch = true
	}

	return
}

func encodeRespToHTTP(w http.ResponseWriter, batch bool, processedResp ...*RPCResponse) {
	if len(processedResp) == 0 {
		// Notification request will be ignored
		w.WriteHeader(http.StatusAccepted)
		return
	}

	var content []byte
	var marshalErr error
	if !batch && len(processedResp) == 1 {
		content, marshalErr = processedResp[0].Marshal()
	} else {
		content, marshalErr = json.Marshal(processedResp)
	}
	if marshalErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(marshalErr.Error()))
	} else {
		w.Write(content)
	}
}

func decodeSingleReq(raw *json.RawMessage) (*RPCRequest, *RPCResponse) {
	req := &RPCRequest{}
	if err := json.Unmarshal(*raw, req); err != nil {
		return nil, NewRPCErrorResponse(nil, InvalidRequestError(err))
	}
	if req.Version != jsonRpcVersion {
		return nil, NewRPCErrorResponse(req.ID, InvalidRequestError(errors.New("invalid jsonrpc version")))
	}
	if req.Method == "" {
		return nil, NewRPCErrorResponse(req.ID, InvalidRequestError(errors.New("invalid method")))
	}
	return req, nil
}
