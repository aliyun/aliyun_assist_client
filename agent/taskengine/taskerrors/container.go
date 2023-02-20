package taskerrors

import (
	"fmt"
)

func NewInvalidContainerIdError() NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "InvalidContainerId",
		cause: fmt.Errorf("The specified containerId is not valid."),
	}
}

func NewInvalidContainerNameError() NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "InvalidContainerName",
		cause: fmt.Errorf("The specified containerName is not valid."),
	}
}

func NewUnsupportedContainerRuntimeError() NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "UnsupportedContainerRuntime",
		cause: fmt.Errorf("The container runtime specified in the containerId is not supported."),
	}
}

func NewContainerNameAndIdNotMatchError(containerId string, expectedName string) NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "ContainerNameAndIdNotMatch",
		cause: fmt.Errorf("The container whose container ID is %s, the name is not %s.", containerId, expectedName),
	}
}

func NewContainerNameDuplicatedError() NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "ContainerNameDuplicated",
		cause: fmt.Errorf("The container for the command to be executed cannot be identified because the instance has a container with the same name."),
	}
}

func NewContainerNotFoundError() NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "ContainerNotFound",
		cause: fmt.Errorf("The specified container does not exist."),
	}
}

func NewContainerStateAbnormalError(currentState string) NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "ContainerStateAbnormal",
		cause: fmt.Errorf("The state of the specified container is abnormal. Current state is %s", currentState),
	}
}

func NewContainerConnectError(cause error) NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: "ContainerConnectFailed",
		cause: fmt.Errorf("Unable to connect to container to invoke command. %w", cause),
	}
}

func NewContainerRuntimeInternalError(cause error) NormalizedExecutionError {
	return &normalizedExecutionErrorImpl{
		code: "ContainerRuntimeInternalError",
		cause: cause,
	}
}

func NewContainerRuntimeTimeoutError(cause error) NormalizedExecutionError {
	return &normalizedExecutionErrorImpl{
		code: "ContainerRuntimeTimeout",
		cause: cause,
	}
}
