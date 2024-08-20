package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/model"
)

func ListContainers(timeout time.Duration, showAllContainers bool) ([]model.Container, error) {
	containerListOptions := types.ContainerListOptions{
		All: showAllContainers,
	}
	if !showAllContainers {
		containerListOptions.Filters = filters.NewArgs(filters.KeyValuePair{
			Key: "status",
			Value: "running",
		})
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	dockerContainers, err := cli.ContainerList(ctx, containerListOptions)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ctxErr, err.Error())
		} else {
			return nil, ctxErr
		}
	}
	if err != nil {
		return nil, err
	}

	var containers []model.Container
	for _, dockerContainer := range dockerContainers {
		container := model.Container{
			Id: dockerContainer.ID,
			RuntimeName: "docker",
			// TODO: FIXME: Use uniform container state representation
			State: strings.ToUpper(dockerContainer.State),
			DataSource: model.ViaDocker,
		}
		if len(dockerContainer.Names) > 0 {
			container.Name = StripAndSelectName(dockerContainer.Names)
		}
		containers = append(containers, container)
	}
	return containers, nil
}
