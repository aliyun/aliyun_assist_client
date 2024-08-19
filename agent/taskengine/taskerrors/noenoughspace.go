package taskerrors

import (
	"fmt"
)

type NoEnoughSpaceError struct {
	cause error
}

func NewNoEnoughSpaceError(cause error) ExecutionError {
	return &NoEnoughSpaceError{
		cause: cause,
	}
}

func (e *NoEnoughSpaceError) Error() string {
	return fmt.Sprintf("NoEnoughSpace: %s", e.cause.Error())
}

func (e *NoEnoughSpaceError) ErrCode() ErrorCode {
	return wrapErrNoEnoughSpace
}
