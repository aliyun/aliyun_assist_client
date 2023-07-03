package container

import (
	"strings"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/cri"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/docker"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
)

type ContainerCommandOptions struct {
	TaskId string
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

func DetectContainerProcessor(options *ContainerCommandOptions) models.TaskProcessor {
	// Detecting on Docker runtime (if available) first, but not when container
	// runtime other than docker specified
	identifierSegments := strings.Split(options.ContainerId, "://")
	if len(identifierSegments) == 1 || identifierSegments[0] == "docker" {
		// Maybe empty if options.ContainerId is just empty string
		dockerContainerId := identifierSegments[0]
		if len(identifierSegments) > 1 {
			dockerContainerId = identifierSegments[1]
		}

		dockerProcessor := &docker.DockerProcessor{
			TaskId: options.TaskId,
			// Fundamental properties of command process
			ContainerId:      dockerContainerId,
			ContainerName:    options.ContainerName,
			CommandType:      options.CommandType,
			Timeout:          options.Timeout,
			WorkingDirectory: options.WorkingDirectory,
			Username:         options.Username,
		}
		if err := docker.CheckDockerProcessor(dockerProcessor); err == nil {
			return dockerProcessor
		} else {
			log.GetLogger().WithFields(logrus.Fields{
				"options": options,
			}).WithError(err).Infoln("Fallback to CRI since Docker runtime with API is not available")
		}
	}

	// Otherwise, fallback to CRI
	return &cri.CRIProcessor{
		TaskId: options.TaskId,
		// Fundamental properties of command process
		ContainerIdentifier: options.ContainerId,
		ContainerName:       options.ContainerName,
		CommandType:         options.CommandType,
		Timeout:             options.Timeout,
	}
}
