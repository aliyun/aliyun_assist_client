package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
)

func createExec(client *dockerclient.Client, container string, config types.ExecConfig, timeout time.Duration) (*types.IDResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := client.ContainerExecCreate(ctx, container, config)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if err != nil {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(err)
		} else {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(ctxErr)
		}
	}
	if err != nil {
		return nil, taskerrors.NewContainerRuntimeInternalError(err)
	}
	return &response, nil
}

func startAndAttachExec(client *dockerclient.Client, execID string, config types.ExecStartCheck, timeout time.Duration) (*types.HijackedResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	hijackedResponse, err := client.ContainerExecAttach(ctx, execID, config)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if err != nil {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(err)
		} else {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(ctxErr)
		}
	}
	if err != nil {
		return nil, taskerrors.NewContainerRuntimeInternalError(err)
	}
	return &hijackedResponse, nil
}

func inspectExec(client *dockerclient.Client, execID string, timeout time.Duration) (*types.ContainerExecInspect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := client.ContainerExecInspect(ctx, execID)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if err != nil {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(err)
		} else {
			return nil, taskerrors.NewContainerRuntimeTimeoutError(ctxErr)
		}
	}
	if err != nil {
		return nil, taskerrors.NewContainerRuntimeInternalError(err)
	}
	return &response, nil
}
