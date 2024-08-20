//go:build linux || freebsd
// +build linux freebsd

package taskerrors

import (
	"errors"
	"strings"
	"syscall"
	"github.com/aliyun/aliyun_assist_client/agent/util/errnoutil"
)

func (e *baseError) Error() string {
	errcodePhrase := e.category
	var errno syscall.Errno
	if errors.As(e.cause, &errno) {
		if errnoPhrase, ok := errnoutil.ErrnoPhrases[errno]; ok {
			errcodePhrase += "." + errnoPhrase
		}
	}

	messages := []string{errcodePhrase}
	if e.Description != "" {
		messages = append(messages, e.Description)
	}
	if e.cause != nil {
		messages = append(messages, e.cause.Error())
	}

	return strings.Join(messages, ": ")
}

func (e *baseError) ErrCode() ErrorCode {
	var errno syscall.Errno
	if errors.As(e.cause, &errno) {
		return ErrorCode(errno)
	}

	return e.categoryCode
}
