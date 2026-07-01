package jsonrpc

import "fmt"

const (
	// Error code defined below are the built-in JSON-RPC errors.
	ParseErrorCode     = -32700
	InvalidRequestCode = -32600
	MethodNotFoundCode = -32601
	InvalidParamsCode  = -32602
	InternalErrorCode  = -32603

	// -32000 to -32099 are reserved for implementation-defined server-errors.
)

// Error code definition reference https://www.jsonrpc.org/specification
func ParseError(err error) *RPCError {
	return &RPCError{
		Code:    ParseErrorCode,
		Message: "Parse error",
		Data:    "An error has occured on the server while parsing the JSON text. " + err.Error(),
	}
}

func InvalidRequestError(err error) *RPCError {
	return &RPCError{
		Code:    InvalidRequestCode,
		Message: "Invalid Request",
		Data:    "The JSON text is not a valid JSON-RPC Request object. " + err.Error(),
	}
}

func MethodNotFoundError(method string) *RPCError {
	return &RPCError{
		Code:    MethodNotFoundCode,
		Message: "Method not found",
		Data:    "The method " + method + " does not exist/is not available",
	}
}

func InvalidParamsError(paramsName string) *RPCError {
	return &RPCError{
		Code:    InvalidParamsCode,
		Message: "Invalid params",
		Data:    "Invalid method parameter(s) " + paramsName,
	}
}

func InternalError(err any) *RPCError {
	return &RPCError{
		Code:    InternalErrorCode,
		Message: "Internal error",
		Data:    fmt.Sprint("Internal JSON-RPC error.", err),
	}
}

func ServerError(code int, message string, data string) *RPCError {
	return &RPCError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}
