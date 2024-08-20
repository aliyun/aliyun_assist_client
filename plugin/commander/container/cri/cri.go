package cri

import (
	"fmt"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	criapis "k8s.io/cri-api/pkg/apis"
	internalapi "k8s.io/cri-api/pkg/apis"
	runtimeapis "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
	libcri "github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/cri"
)

type containerConnection struct {
	runtimeService criapis.RuntimeService
	containerId    string
	containerName  string
}

type filterFunc func([]*runtimeapis.Container) []*runtimeapis.Container

func DetectCriProcessor(logger logrus.FieldLogger, runtimeType string, options *model.ContainerCommandOptions) (*CRIProcessor, error) {
	criRuntimeServices, criEndpoints, err := getCriRuntimeService(runtimeType)
	if err != nil {
		return nil, err
	}
	connection, err := findTargetContainer(criRuntimeServices, criEndpoints, options.ContainerId, options.ContainerName)
	if err != nil {
		return nil, err
	}
	return NewProcessor(options, connection), nil
}

// GetCriRuntimeService get available cri runtime services and their endpoints
func getCriRuntimeService(runtimeType string) ([]internalapi.RuntimeService, []libcri.RuntimeEndpoint, error) {
	var runtimeEndpoints []libcri.RuntimeEndpoint
	if runtimeType != "" {
		supportedEndpoints, ok := libcri.RuntimeName2Endpoints[runtimeType]
		if !ok {
			validationErr := taskerrors.NewUnsupportedContainerRuntimeError()
			return nil, nil, validationErr
		}
		runtimeEndpoints = supportedEndpoints
	} else {
		runtimeEndpoints = libcri.PrecedentRuntimeEndpoints
	}

	var runtimeServices []internalapi.RuntimeService
	var validEndpoints []libcri.RuntimeEndpoint
	connectTimeout := 10 * time.Second
	for _, endpoint := range runtimeEndpoints {
		service, err := remote.NewRemoteRuntimeService(endpoint.Endpoint, connectTimeout)
		if err != nil {
			log.GetLogger().WithError(err).Warningf("Failed to connect %s runtime via %s", endpoint.RuntimeName, endpoint.Endpoint)
			continue
		}
		runtimeServices = append(runtimeServices, service)
		validEndpoints = append(validEndpoints, endpoint)
	}

	return runtimeServices, validEndpoints, nil
}

// findTargetContainer find the taget container by available runtime services
func findTargetContainer(runtimeServices []internalapi.RuntimeService, endpoints []libcri.RuntimeEndpoint, containerId, containerName string) (*containerConnection, error) {
	conditions := &runtimeapis.ContainerFilter{}
	var filterByName filterFunc = nil
	if containerId != "" {
		conditions.Id = containerId
	}
	if containerName != "" {
		filterByName = func(before []*runtimeapis.Container) []*runtimeapis.Container {
			after := make([]*runtimeapis.Container, 0, len(before))
			for _, container := range before {
				if container.Metadata != nil && container.Metadata.Name == containerName {
					after = append(after, container)
				}
			}
			return after
		}
	}
	availableRuntimeServiceCount := 0
	for idx, service := range runtimeServices {
		containers, err := service.ListContainers(conditions)
		if err != nil {
			log.GetLogger().WithError(err).Warningf("Failed to list containers matching specified conditions on %s runtime via %s", endpoints[idx].RuntimeName, endpoints[idx].Endpoint)
			continue
		}
		availableRuntimeServiceCount += 1

		if filterByName != nil {
			containersOfName := filterByName(containers)
			// Special handling when both containerId and containerName
			// specified: if container found by id is filtered out by name,
			// ContainerNameAndIdNotMatch error should be reported immediately.
			if containerId != "" && len(containers) > 0 && len(containersOfName) == 0 {
				return nil, taskerrors.NewContainerNameAndIdNotMatchError(containerId, containerName)
			}

			containers = containersOfName
		}
		if len(containers) == 0 {
			log.GetLogger().WithFields(logrus.Fields{
				"containerId":   containerId,
				"containerName": containerName,
			}).WithError(err).Warningf("Cannot find container matching specified conditions on %s runtime via %s", endpoints[idx].RuntimeName, endpoints[idx].Endpoint)
			continue
		}
		// As per the OCI runtime spec, "This (container id) MUST be unique
		// across all containers on this host." Although the situation that
		// multiple container runtime installed on the same host is not
		// mentioned in the spec, but such should be extremely rare in
		// production environment. So, further detection on other runtimes
		// should be able to be safely ignored, and we can think we have found
		// the right answer.
		return determineContainer(service, containers)
	}

	if availableRuntimeServiceCount == 0 {
		return nil, taskerrors.NewContainerConnectError(fmt.Errorf("No available container runtime to find container for executing command"))
	}
	return nil, taskerrors.NewContainerNotFoundError()
}

func determineContainer(service criapis.RuntimeService, containers []*runtimeapis.Container) (*containerConnection, error) {
	if len(containers) > 1 {
		return nil, taskerrors.NewContainerNameDuplicatedError()
	}

	connection := &containerConnection{
		runtimeService: service,
		containerId:    containers[0].Id,
	}
	if containers[0].Metadata != nil {
		connection.containerName = containers[0].Metadata.Name
	}
	// Codes below SHOULD always return connection object to indicate the
	// container matching specified attributes has been found successfully
	if containers[0].State != runtimeapis.ContainerState_CONTAINER_RUNNING {
		return connection, taskerrors.NewContainerStateAbnormalError(containers[0].State.String())
	}

	return connection, nil
}
