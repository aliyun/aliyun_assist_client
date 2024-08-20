package taskerrors

import (
	"strings"
)

func (e *baseError) Error() string {
	messages := []string{e.category}
	if e.Description != "" {
		messages = append(messages, e.Description)
	}
	if e.cause != nil {
		messages = append(messages, e.cause.Error())
	}

	return strings.Join(messages, ": ")
}

func (e *baseError) ErrCode() ErrorCode {
	return e.categoryCode
}
