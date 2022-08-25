package taskerrors

type InvalidSettingError interface {
	error

	Name() string
	ShortMessage() string
}

type ExecutionError interface {
	error

	Code() ErrorCode
}
