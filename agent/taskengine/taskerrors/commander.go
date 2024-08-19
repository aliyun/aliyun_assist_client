package taskerrors

import (
	"fmt"
	"strings"
)

type CommanderError struct {
	Code    string
	SubCode string
	Message string
}

// Code
const (
	errCodeCommanderNotFound = "CommanderNotFound"
	errCodeCommanderStart    = "StartCommanderFailed"
	errCodeCommanderTask     = "CommanderTaskError"
	errCodeIpcError          = "IpcError"
)

// SubCode
const (
	// CommanderNotFound
	errSubCodeLoadCommanderFail = "LoadCommanderFailed"

	// StartCommanderFailed
	errSubCodeClientNotCreated  = "ClientNotCreated"
	errSubCodeCreateClientFail  = "CreateClientFailed"
	errSubCodeStartProcessFail  = "StartProcessFailed"
	errSubCodeAttachProcessFail = "AttachProcessFailed"

	// CommanderTaskError
	errSubCodeStatusUpdateTimeout = "StatusUpdateTimeout"
	errSubCodeCommanderExit       = "CommanderExit"

	// IpcError
	errSubCodeRequestError = "RequestError"
)

func NewCommanderError(code, subcode, message string) *CommanderError {
	return &CommanderError{
		Code:    code,
		SubCode: subcode,
		Message: message,
	}
}

func (e *CommanderError) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.Code, e.SubCode, e.Message)
}

func (e *CommanderError) SubMessage() string {
	return fmt.Sprintf("%s: %s", e.SubCode, e.Message)
}

func NewLoadCommanderFailedError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderNotFound,
		SubCode: errSubCodeLoadCommanderFail,
		Message: strings.Join(message, ". "),
	}
}

func NewClientNotCreatedError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderStart,
		SubCode: errSubCodeClientNotCreated,
		Message: strings.Join(message, ". "),
	}
}

func NewCreateClientFailedError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderStart,
		SubCode: errSubCodeCreateClientFail,
		Message: strings.Join(message, ". "),
	}
}

func NewStartProcessFailedError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderStart,
		SubCode: errSubCodeStartProcessFail,
		Message: strings.Join(message, ". "),
	}
}

func NewAttachProcessFailedError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderStart,
		SubCode: errSubCodeAttachProcessFail,
		Message: strings.Join(message, ". "),
	}
}

func NewIpcRequestFailedError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeIpcError,
		SubCode: errSubCodeRequestError,
		Message: strings.Join(message, ". "),
	}
}

func NewStatusUpdateTimeoutError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderTask,
		SubCode: errSubCodeStatusUpdateTimeout,
		Message: strings.Join(message, ". "),
	}
}

func NewCommanderExitError(message ...string) *CommanderError {
	return &CommanderError{
		Code:    errCodeCommanderTask,
		SubCode: errSubCodeCommanderExit,
		Message: strings.Join(message, ". "),
	}
}
