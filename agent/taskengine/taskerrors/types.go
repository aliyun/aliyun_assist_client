package taskerrors

type InvalidSettingError interface {
	error

	Name() string
	ShortMessage() string
	Unwrap() error
}

type ExecutionError interface {
	error

	Code() ErrorCode
}
