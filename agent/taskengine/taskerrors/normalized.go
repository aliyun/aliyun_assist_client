package taskerrors

import "fmt"

type NormalizedValidationError interface {
	error

	Param() string
	Value() string
}

type normalizedValidationErrorImpl struct {
	category string
	cause error
}

func NormalizeValidationError(category string, cause error) NormalizedValidationError {
	return &normalizedValidationErrorImpl{
		category: category,
		cause: cause,
	}
}

func (nve *normalizedValidationErrorImpl) Error() string {
	if nve.cause != nil {
		return fmt.Sprintf("%s: %s", nve.category, nve.cause.Error())
	} else {
		return nve.category
	}
}

func (nve *normalizedValidationErrorImpl) Param() string {
	return nve.category
}

func (nve *normalizedValidationErrorImpl) Value() string {
	return nve.Error()
}

type NormalizedExecutionError interface {
	error

	Code() string
	Description() string
}

type normalizedExecutionErrorImpl struct {
	code string
	cause error
}

func NormalizeExecutionError(code string, cause error) NormalizedExecutionError {
	return &normalizedExecutionErrorImpl{
		code: code,
		cause: cause,
	}
}

func (nve *normalizedExecutionErrorImpl) Error() string {
	if nve.cause != nil {
		return fmt.Sprintf("%s: %s", nve.code, nve.cause.Error())
	} else {
		return nve.code
	}
}

func (nve *normalizedExecutionErrorImpl) Code() string {
	return nve.code
}

func (nve *normalizedExecutionErrorImpl) Description() string {
	return nve.Error()
}
