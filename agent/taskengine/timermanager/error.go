package timermanager

import "fmt"

type CronParameterError struct {
	code string
	message string
}

func newCronParameterError(code string, message string) CronParameterError {
	return CronParameterError{
		code: code,
		message: message,
	}
}

func (e *CronParameterError) Code() string {
	return e.code
}

func (e *CronParameterError) Message() string {
	return e.message
}

func (e CronParameterError) Error() string {
	return fmt.Sprintf("%s: %s", e.code, e.message)
}
