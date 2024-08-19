package container

import (
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/cri"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/docker"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
)

func DetectContainerProcessor(options *model.ContainerCommandOptions) (model.TaskProcessor, error) {
	logger := log.GetLogger().WithField("options", options)
	var runtimeType, containerId string
	identifierSegments := strings.Split(options.ContainerId, "://")
	if len(identifierSegments) == 1 {
		containerId = identifierSegments[0]
	} else {
		runtimeType = identifierSegments[0]
		containerId = identifierSegments[1]
	}
	if len(identifierSegments) > 1 && containerId == "" {
		// Or the type of CRI-compatible container runtime has been extracted from
		// container identifier in format <type>://<container-id>
		return nil, taskerrors.NewInvalidContainerIdError()
	}
	options.ContainerId = containerId
	var dockerProcessor *docker.DockerProcessor
	var criProcessor *cri.CRIProcessor
	var err error
	logger.Infof("runtimeType: %s, containerId: %s", runtimeType, containerId)
	if runtimeType == "" || runtimeType == "docker" {
		// Maybe empty if options.ContainerId is just empty string
		dockerProcessor, err = docker.DetectDockerProcessor(logger, options)
		if err != nil {
			logger.WithError(err).Error("Detect docker processor failed")
		}
	}
	if dockerProcessor == nil {
		criProcessor, err = cri.DetectCriProcessor(logger, runtimeType, options)
		if err != nil {
			logger.WithError(err).Error("Detect cri processor failed")
		}
		return criProcessor, err
	}
	return dockerProcessor, nil
}
