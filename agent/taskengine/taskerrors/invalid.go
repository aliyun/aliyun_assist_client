package taskerrors

import (
	"errors"
	"fmt"
)

type settingError struct {
	name string
	shortMessage string
	message string
	cause error
}

func (e *settingError) Name() string {
	return e.name
}

func (e *settingError) ShortMessage() string {
	return e.shortMessage
}

func (e *settingError) Error() string {
	return e.message
}

func (e *settingError) Unwrap() error {
	return e.cause
}

func NewInvalidUsernameOrPasswordError(cause error, shortMessage string) InvalidSettingError {
	return &settingError{
		name: "UsernameOrPasswordInvalid",
		shortMessage: shortMessage,
		message: fmt.Sprintf("%s: %s", "UsernameOrPasswordInvalid", shortMessage),
		cause: cause,
	}
}

func NewHomeDirectoryNotAvailableError(cause error) InvalidSettingError {
	return &settingError{
		name: "homeDir",
		shortMessage: "HomeDirectoryNotAvailable",
		message: fmt.Sprintf("HomeDirectoryNotAvailable: Failed to detect home directory of specified user: %s", cause.Error()),
		cause: cause,
	}
}

func NewWorkingDirectoryNotExistError(workingDir string) InvalidSettingError {
	return &settingError{
		name: "workingDirectory",
		shortMessage: "WorkingDirectoryNotExist",
		message: fmt.Sprintf("WorkingDirectoryNotExist: %s", workingDir),
		cause: fmt.Errorf("%s does not exist", workingDir),
	}
}

func NewDefaultWorkingDirectoryNotAvailableError(message string) InvalidSettingError {
	return &settingError{
		name: "workingDirectory",
		shortMessage: "DefaultWorkingDirectoryNotAvailable",
		message: fmt.Sprintf("DefaultWorkingDirectoryNotAvailable: %s", message),
		cause: errors.New(message),
	}
}

func NewInvalidEnvironmentParameterError(message string) InvalidSettingError {
	return &settingError{
		name: "InvalidEnvironmentParameter",
		shortMessage: message,
		message: message,
		cause: nil,
	}
}
