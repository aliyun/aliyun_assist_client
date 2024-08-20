package docker

import (
	"context"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"

	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
)

func DetectDockerProcessor(logger logrus.FieldLogger, options *model.ContainerCommandOptions) (*DockerProcessor, error) {
	dockerClient, err := newDockerClient()
	if err != nil {
		return nil, err
	}
	container, err := findTargetContainer(dockerClient, options.ContainerId, options.ContainerName)
	if err != nil {
		dockerClient.Close()
		return nil, err
	}
	return NewProcessor(options, dockerClient, container), nil
}

// newDockerClient create a docker client
func newDockerClient() (*dockerclient.Client, error) {
	var err error
	var client *dockerclient.Client
	client, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, taskerrors.NewContainerConnectError(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = client.Ping(ctx)
	if err != nil {
		return nil, taskerrors.NewContainerConnectError(err)
	}
	return client, nil
}

// findTargetContainer find the target container by docker client
func findTargetContainer(dockerClient *dockerclient.Client, containerId, containerName string) (*types.Container, error) {
	// Three situations MUST be carefully handled:
	// 1. Only container id specified fortunately.
	// 2. Only container name specified, which should be validated to be conform
	//    to the requirement in API reference.
	// 3. Both specified. Not only the format of container name should be
	// validated at first, but the name itself needs to be checked if it belongs
	// to the container.
	if containerName != "" {
		if !containerNameValidator.MatchString(containerName) {
			return nil, taskerrors.NewInvalidContainerNameError()
		}
	}

	containerListFilterKVs := make([]filters.KeyValuePair, 0, 2)
	// Although the original ContainerId parameter supports Kubernetes's special
	// <runtime>://<container-id> format, but don't worry here. runtime prefix
	// has been correctly stripped in caller function.
	if containerId != "" {
		containerListFilterKVs = append(containerListFilterKVs, filters.KeyValuePair{
			Key:   "id",
			Value: containerId,
		})
	} else if containerName != "" {
		containerListFilterKVs = append(containerListFilterKVs, filters.KeyValuePair{
			Key:   "name",
			Value: containerName,
		})
	}
	containerListOptions := types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(containerListFilterKVs...),
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	foundContainers, err := dockerClient.ContainerList(ctx, containerListOptions)
	if ctxErr := ctx.Err(); ctxErr != nil {
		log.GetLogger().WithError(err).Warningf("Failed to list docker containers matching filter %v", containerListFilterKVs)
		if err != nil {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(err)
		} else {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(ctxErr)
		}
	}
	if err != nil {
		log.GetLogger().WithError(err).Warningf("Failed to list docker containers matching filter %v", containerListFilterKVs)
		return nil, taskerrors.NewContainerRuntimeInternalError(err)
	}
	if len(foundContainers) == 0 {
		return nil, taskerrors.NewContainerNotFoundError()
	}

	if containerId != "" && containerName != "" {
		// Container names in output of `docker ps`, `docker container list` or
		// `docker list` are not prefixed by a slash '/'. HOWEVER container
		// names got from docker's client.ContainerList() does have it, i.e.,
		// what you see is NOT what you get. Looking inside the source of docker
		// CLI, magics are applied to these names before presented to Muggles,
		// and that's what we need in our humble pre-processing phrase before
		// naive container filtering by name.
		// See link below for detail implementation in docker CLI:
		// https://github.com/docker/cli/blob/67cc8b1fd88aea06690eaf3e5d56acd68a0178d2/cli/command/formatter/container.go#L125-L148
		//
		// And I know, I know passing both container id and name to Docker's
		// ContainerList API would be a simpler solution. But it is a little
		// expensive than filtering by ourselves. Consideration is still open.
		prefixedContainerName := containerName
		if prefixedContainerName[0] != '/' {
			prefixedContainerName = "/" + prefixedContainerName
		}

		filteredContainers := make([]types.Container, 0, len(foundContainers))
		for _, container := range foundContainers {
			for _, name := range container.Names {
				if name == prefixedContainerName {
					filteredContainers = append(filteredContainers, container)
					break
				}
			}
		}

		if len(filteredContainers) == 0 {
			return nil, taskerrors.NewContainerNameAndIdNotMatchError(containerId, containerName)
		} else {
			foundContainers = filteredContainers
		}
	}

	if len(foundContainers) == 0 {
		return nil, taskerrors.NewContainerNotFoundError()
	}
	if len(foundContainers) > 1 {
		return nil, taskerrors.NewContainerNameDuplicatedError()
	}

	// TODO: FIXME: Use uniform container state representation and convert to it
	canonicalizedContainerState := strings.ToUpper(foundContainers[0].State)
	if canonicalizedContainerState != "RUNNING" {
		return nil, taskerrors.NewContainerStateAbnormalError(canonicalizedContainerState)
	}

	return &foundContainers[0], nil
}
