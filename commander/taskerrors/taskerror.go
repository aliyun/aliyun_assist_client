package taskerrors

import (
	"fmt"
)

const (
	errCommanderCode = "CommanderError"
)

const (
	errSubCodeSubmissionDuplicate   = "SubmissionDuplicate"
	errSubCodeSubmissionNotExists   = "SubmissionNotExists"
	errSubCodeSubmissionTypeInvalid = "SubmissionTypeInvalid"
	errSubCodeCreatePipeError       = "CreatePipeError"
	errSubCodePrepareContentInvalid = "PrepareContentInvalid"
	errSubCodeSubmissionRunError    = "SubmissionRunError"
	errSubCodeGeneralError          = "GeneralError"
)

type TaskError struct {
	ErrorCode    string
	ErrorSubCode string
	ErrorMessage string
}

func (e *TaskError) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.ErrorCode, e.ErrorSubCode, e.ErrorMessage)
}

func (e *TaskError) String() string {
	return fmt.Sprintf("%s.%s: %s", e.ErrorCode, e.ErrorSubCode, e.ErrorMessage)
}

func NewSubmissionDuplicateError() *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodeSubmissionDuplicate,
		ErrorMessage: "Submission duplicate",
	}
}

func NewSubmissionNotExistsError() *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodeSubmissionNotExists,
		ErrorMessage: "Submission not exists",
	}
}

func NewSubmissionTypeInvalidError() *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodeSubmissionTypeInvalid,
		ErrorMessage: "Submission type invalid",
	}
}

func NewCreatePipeError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodeCreatePipeError,
		ErrorMessage: cause.Error(),
	}
}

func NewPrepareContentInvalidError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodePrepareContentInvalid,
		ErrorMessage: cause.Error(),
	}
}

func NewSubmissionRunError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodeSubmissionRunError,
		ErrorMessage: cause.Error(),
	}
}

func NewTaskError(errorCode, errorSubCode, errorMessage string) *TaskError {
	return &TaskError{
		ErrorCode:    errorCode,
		ErrorSubCode: errorSubCode,
		ErrorMessage: errorMessage,
	}
}

func NewGeneralError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCommanderCode,
		ErrorSubCode: errSubCodeGeneralError,
		ErrorMessage: cause.Error(),
	}
}
