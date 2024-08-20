package taskerrors

import (
	"fmt"
	"strconv"
)

// ErrorCode defines and MUST contain all error codes that will be reported
// as failure
type ErrorCode int

const (
	// Positive value is reserved for syscall errno on *nix and API error code
	// on Windows

	wrapErrGetScriptPathFailed ErrorCode = -(1 + iota)
	wrapErrUnknownCommandType
	WrapErrBase64DecodeFailed
	wrapErrSaveScriptFileFailed
	wrapErrSetExecutablePermissionFailed
	wrapErrSetWindowsPermissionFailed
	WrapErrExecuteScriptFailed
	wrapErrNoEnoughSpace
	wrapErrScriptFileExisted
	wrapErrPowershellNotFound
	wrapErrSystemDefaultShellNotFound
	WrapErrResolveEnvironmentParameterFailed

	WrapGeneralError
	wrapErrConnectContainerRuntimeFailed
	wrapErrNoAvailableContainerRuntime
	wrapErrContainerRuntimeInternalFailed
	wrapErrContainerNotFoundById
	wrapErrManyContainersFoundById
	wrapErrContainerNotRunning

	WrapErrCreatePipeFailed
	WrapErrCreateProcessCollectionFailed
	WrapCommanderError
	WrapErrServerResponseError // Backend server Response errorCode for api task/xxx
)

func (c ErrorCode) String() string {
	return strconv.Itoa(int(c))
}

type baseError struct {
	categoryCode ErrorCode
	category string
	Description string
	cause error
}

func (e *baseError) Unwrap() error {
	return e.cause
}

func NewGetScriptPathError(cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrGetScriptPathFailed,
		category: "GetScriptPathFailed",
		cause: cause,
	}
}

func NewUnknownCommandTypeError() ExecutionError {
	return &baseError{
		categoryCode: wrapErrUnknownCommandType,
		category: "UnknownCommandType",
		cause: nil,
	}
}

func NewScriptFileExistedError(savePath string, cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrScriptFileExisted,
		category: "ScriptFileExisted",
		Description: fmt.Sprintf("Saving script to %s failed", savePath),
		cause: cause,
	}
}

func NewSaveScriptFileError(cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrSaveScriptFileFailed,
		category: "SaveScriptFileFailed",
		cause: cause,
	}
}

func NewSetExecutablePermissionError(cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrSetExecutablePermissionFailed,
		category: "SetExecutablePermissionFailed",
		Description: "Failed to set executable permission of shell script",
		cause: cause,
	}
}

func NewExecuteScriptError(cause error) ExecutionError {
	return &baseError{
		categoryCode: WrapErrExecuteScriptFailed,
		category: "ExecuteScriptFailed",
		cause: cause,
	}
}

func NewSetWindowsPermissionError(cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrSetWindowsPermissionFailed,
		category: "SetWindowsPermissionFailed",
		Description: "Failed to set permission of script on Windows",
		cause: cause,
	}
}

func NewSystemDefaultShellNotFoundError(cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrSystemDefaultShellNotFound,
		category: "SystemDefaultShellNotFound",
		cause: cause,
	}
}

func NewPowershellNotFoundError(cause error) ExecutionError {
	return &baseError{
		categoryCode: wrapErrPowershellNotFound,
		category: "PowershellNotFound",
		cause: cause,
	}
}

func NewResolvingInstanceNameError(cause error) ExecutionError {
	return &baseError{
		categoryCode: WrapErrResolveEnvironmentParameterFailed,
		category: "ResolvingInstanceNameFailed",
		cause: cause,
	}
}

func NewCreateProcessCollectionError(cause error) ExecutionError {
	return &baseError{
		categoryCode: WrapGeneralError,
		category: "CreateProcessCollectionFailed",
		cause: cause,
	}
}

func NewCreatePipeError(cause error) ExecutionError {
	return &baseError{
		categoryCode: WrapErrCreatePipeFailed,
		category: "CreatePipeFailed",
		cause: cause,
	}
}

func NewServerResponseError(cause error) ExecutionError {
	return &baseError{
		categoryCode: WrapErrServerResponseError,
		category: "ServerResponseError",
		cause: cause,
	}
}
