package jsonrpc

import (
	"errors"
	"fmt"
	"testing"
)

func TestParseError(t *testing.T) {
	err := errors.New("test parse error")
	rpcErr := ParseError(err)
	if rpcErr.Code != ParseErrorCode {
		t.Errorf("Expected code %d, got %d", ParseErrorCode, rpcErr.Code)
	}
	if rpcErr.Message != "Parse error" {
		t.Errorf("Expected message 'Parse error', got '%s'", rpcErr.Message)
	}
	expectedData := "An error has occured on the server while parsing the JSON text. test parse error"
	if rpcErr.Data != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, rpcErr.Data)
	}
}

func TestInvalidRequestError(t *testing.T) {
	err := errors.New("test invalid request error")
	rpcErr := InvalidRequestError(err)
	if rpcErr.Code != InvalidRequestCode {
		t.Errorf("Expected code %d, got %d", InvalidRequestCode, rpcErr.Code)
	}
	if rpcErr.Message != "Invalid Request" {
		t.Errorf("Expected message 'Invalid Request', got '%s'", rpcErr.Message)
	}
	expectedData := "The JSON text is not a valid JSON-RPC Request object. test invalid request error"
	if rpcErr.Data != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, rpcErr.Data)
	}
}

func TestMethodNotFoundError(t *testing.T) {
	method := "testMethod"
	rpcErr := MethodNotFoundError(method)
	if rpcErr.Code != MethodNotFoundCode {
		t.Errorf("Expected code %d, got %d", MethodNotFoundCode, rpcErr.Code)
	}
	if rpcErr.Message != "Method not found" {
		t.Errorf("Expected message 'Method not found', got '%s'", rpcErr.Message)
	}
	expectedData := "The method testMethod does not exist/is not available"
	if rpcErr.Data != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, rpcErr.Data)
	}
}

func TestInvalidParamsError(t *testing.T) {
	rpcErr := InvalidParamsError("paramA")
	if rpcErr.Code != InvalidParamsCode {
		t.Errorf("Expected code %d, got %d", InvalidParamsCode, rpcErr.Code)
	}
	if rpcErr.Message != "Invalid params" {
		t.Errorf("Expected message 'Invalid params', got '%s'", rpcErr.Message)
	}
	expectedData := "Invalid method parameter(s) paramA"
	if rpcErr.Data != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, rpcErr.Data)
	}
}

func TestInternalError(t *testing.T) {
	err := "test internal error"
	rpcErr := InternalError(err)
	if rpcErr.Code != InternalErrorCode {
		t.Errorf("Expected code %d, got %d", InternalErrorCode, rpcErr.Code)
	}
	if rpcErr.Message != "Internal error" {
		t.Errorf("Expected message 'Internal error', got '%s'", rpcErr.Message)
	}
	expectedData := fmt.Sprint("Internal JSON-RPC error.", err)
	if rpcErr.Data != expectedData {
		t.Errorf("Expected data '%s', got '%s'", expectedData, rpcErr.Data)
	}
}

func TestServerError(t *testing.T) {
	code := -32000
	message := "Test server error"
	data := "Server error data"
	rpcErr := ServerError(code, message, data)
	if rpcErr.Code != code {
		t.Errorf("Expected code %d, got %d", code, rpcErr.Code)
	}
	if rpcErr.Message != message {
		t.Errorf("Expected message '%s', got '%s'", message, rpcErr.Message)
	}
	if rpcErr.Data != data {
		t.Errorf("Expected data '%s', got '%s'", data, rpcErr.Data)
	}
}
