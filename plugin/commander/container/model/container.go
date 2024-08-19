package model

type ContainerCommandOptions struct {
	SubmissionID string
	// Fundamental properties of command process
	ContainerId   string
	ContainerName string // may be filled when only container id specified
	CommandType   string
	// CommandContent is not needed here
	Timeout int
	// Additional execution attributes supported by docker
	WorkingDirectory string
	Username         string
}

const (
	CommanderName = "ACS-ECS-ContainerCommander"

	// supported grpc api version, currently only supports version v1
	CommanderSupportedApiVersion = "v1"
)
