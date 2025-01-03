package httpbase

import (
	"fmt"
)

type StatusCodeError struct {
	statusCode int
}

func NewStatusCodeError(code int) *StatusCodeError {
	return &StatusCodeError{
		statusCode: code,
	}
}

func (e *StatusCodeError) Error() string {
	return fmt.Sprintf("unexpected HTTP(S) status code %d", e.statusCode)
}

func (e *StatusCodeError) StatusCode() int {
	return e.statusCode
}
