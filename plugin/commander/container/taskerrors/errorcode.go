package taskerrors

import (
	"fmt"
)

type TaskError struct {
	ErrorCode    string
	ErrorSubCode string
	ErrorMessage string
}

const (
	errCodeCommanderTask = "CommanderTaskError"
)

const (
	errSubCodeInvalidContainerId            = "InvalidContainerId"
	errSubCodeInvalidContainerName          = "InvalidContainerName"
	errSubCodeUnsupportedContainerRuntime   = "UnsupportedContainerRuntime"
	errSubCodeContainerNameAndIdNotMatch    = "ContainerNameAndIdNotMatch"
	errSubCodeContainerNameDuplicated       = "ContainerNameDuplicated"
	errSubCodeContainerNotFound             = "ContainerNotFound"
	errSubCodeContainerStateAbnormal        = "ContainerStateAbnormal"
	errSubCodeContainerConnectFailed        = "ContainerConnectFailed"
	errSubCodeSubmissionInvalid             = "SubmissionInvalid"
	errSubCodeContainerRuntimeInternalError = "ContainerRuntimeInternalError"
	errSubCodeContainerRuntimeTimeout       = "ContainerRuntimeTimeout"
	errSubCodeCreatePipeError               = "CreatePipeError"
	errSubCodeGeneralError                  = "GeneralError"
)

func (e *TaskError) Error() string {
	return fmt.Sprintf("%s.%s: %s", e.ErrorCode, e.ErrorSubCode, e.ErrorMessage)
}

func (e *TaskError) String() string {
	return fmt.Sprintf("%s.%s: %s", e.ErrorCode, e.ErrorSubCode, e.ErrorMessage)
}

func NewInvalidContainerIdError() *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeInvalidContainerId,
		ErrorMessage: "The specified containerId is not valid.",
	}
}

func NewInvalidContainerNameError() *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeInvalidContainerName,
		ErrorMessage: "The specified containerName is not valid.",
	}
}

func NewUnsupportedContainerRuntimeError() *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeUnsupportedContainerRuntime,
		ErrorMessage: "The container runtime specified in the containerId is not supported.",
	}
}

func NewContainerNameAndIdNotMatchError(containerId string, expectedName string) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerNameAndIdNotMatch,
		ErrorMessage: fmt.Sprintf("The container whose container ID is %s, the name is not %s.", containerId, expectedName),
	}
}

func NewContainerNameDuplicatedError() *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerNameDuplicated,
		ErrorMessage: "The container for the command to be executed cannot be identified because the instance has a container with the same name.",
	}
}

func NewContainerNotFoundError() *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerNotFound,
		ErrorMessage: "The specified container does not exist.",
	}
}

func NewContainerStateAbnormalError(currentState string) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerStateAbnormal,
		ErrorMessage: fmt.Sprintf("The state of the specified container is abnormal. Current state is %s", currentState),
	}
}

func NewContainerConnectError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerConnectFailed,
		ErrorMessage: fmt.Sprintf("Unable to connect to container to invoke command. %v", cause),
	}
}

func NewSubmissionInvalidError(message string) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeSubmissionInvalid,
		ErrorMessage: message,
	}
}

func NewContainerRuntimeInternalError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerRuntimeInternalError,
		ErrorMessage: cause.Error(),
	}
}

func NewContainerRuntimeTimeoutError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeContainerRuntimeTimeout,
		ErrorMessage: cause.Error(),
	}
}

func NewCreatePipeError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeCreatePipeError,
		ErrorMessage: cause.Error(),
	}
}

func NewGeneralExecutionError(cause error) *TaskError {
	return &TaskError{
		ErrorCode:    errCodeCommanderTask,
		ErrorSubCode: errSubCodeGeneralError,
		ErrorMessage: cause.Error(),
	}
}
