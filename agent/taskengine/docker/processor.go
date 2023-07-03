package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerclient "github.com/docker/docker/client"

	dockerutil "github.com/aliyun/aliyun_assist_client/agent/container/docker"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

const (
	defaultTimeout       time.Duration = 10 * time.Second
	maxInspectionRetries               = 5
)

var (
	ErrInconsistentExecProcessState = errors.New("Exec session in the container terminated but process still running")

	containerNameValidator = regexp.MustCompile(`^/?[a-zA-Z0-9][a-zA-Z0-9_.-]+$`)
)

type DockerProcessor struct {
	TaskId string
	// Fundamental properties of command process
	ContainerId    string
	ContainerName  string // may be filled when only container id specified
	CommandType    string
	CommandContent string
	Timeout        int
	// Additional execution attributes supported by docker
	WorkingDirectory string
	Username         string

	client          *dockerclient.Client
	foundContainers []types.Container
	// stripped and selected name for the container found
	foundContainerName string
	cancel             context.CancelFunc
}

// CheckDockerProcessor performs some pre-checking logics to determine whether
// Docker runtime should be used for execution.
func CheckDockerProcessor(p *DockerProcessor) error {
	// Three situations MUST be carefully handled:
	// 1. Only container id specified fortunately.
	// 2. Only container name specified, which should be validated to be conform
	//    to the requirement in API reference.
	// 3. Both specified. Not only the format of container name should be
	// validated at first, but the name itself needs to be checked if it belongs
	// to the container.
	if p.ContainerName != "" {
		if !containerNameValidator.MatchString(p.ContainerName) {
			return taskerrors.NewInvalidContainerNameError()
		}
	}

	var err error
	p.client, err = dockerclient.NewClientWithOpts(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return taskerrors.NewContainerConnectError(err)
	}

	containerListFilterKVs := make([]filters.KeyValuePair, 0, 2)
	// Although the original ContainerId parameter supports Kubernetes's special
	// <runtime>://<container-id> format, but don't worry here. runtime prefix
	// has been correctly stripped in caller function.
	if p.ContainerId != "" {
		containerListFilterKVs = append(containerListFilterKVs, filters.KeyValuePair{
			Key:   "id",
			Value: p.ContainerId,
		})
	} else if p.ContainerName != "" {
		containerListFilterKVs = append(containerListFilterKVs, filters.KeyValuePair{
			Key:   "name",
			Value: p.ContainerName,
		})
	}
	containerListOptions := types.ContainerListOptions{
		All:     true,
		Filters: filters.NewArgs(containerListFilterKVs...),
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	p.foundContainers, err = p.client.ContainerList(ctx, containerListOptions)
	if ctxErr := ctx.Err(); ctxErr != nil {
		if err != nil {
			return taskerrors.NewContainerRuntimeTimeoutError(err)
		} else {
			return taskerrors.NewContainerRuntimeTimeoutError(ctxErr)
		}
	}
	if err != nil {
		return taskerrors.NewContainerRuntimeInternalError(err)
	}
	if len(p.foundContainers) == 0 {
		return taskerrors.NewContainerNotFoundError()
	}

	if p.ContainerId != "" && p.ContainerName != "" {
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
		prefixedContainerName := p.ContainerName
		if prefixedContainerName[0] != '/' {
			prefixedContainerName = "/" + prefixedContainerName
		}

		filteredContainers := make([]types.Container, 0, len(p.foundContainers))
		for _, container := range p.foundContainers {
			for _, name := range container.Names {
				if name == prefixedContainerName {
					filteredContainers = append(filteredContainers, container)
					break
				}
			}
		}

		if len(filteredContainers) == 0 {
			return taskerrors.NewContainerNameAndIdNotMatchError(p.ContainerId, p.ContainerName)
		} else {
			p.foundContainers = filteredContainers
		}
	}

	if len(p.foundContainers) > 1 {
		return taskerrors.NewContainerNameDuplicatedError()
	}

	// I know, I know the complete pre-checking procedure needs more validation
	// to find the correct container. But this function is mostly used to
	// determine whether Docker runtime should be chosen to connect container,
	// so further checking is postponed to DockerProcessor.PreCheck() method.
	return nil
}

// PreCheck method of DockerProcessor struct continues to perform pre-checking
// actions left by CheckDockerProcessor() function.
func (p *DockerProcessor) PreCheck() (string, error) {
	if len(p.foundContainers) == 0 {
		validationErr := taskerrors.NewContainerNotFoundError()
		return validationErr.Param(), validationErr
	}

	if len(p.foundContainers) > 1 {
		validationErr := taskerrors.NewContainerNameDuplicatedError()
		return validationErr.Param(), validationErr
	}

	foundContainer := p.foundContainers[0]
	p.foundContainerName = dockerutil.StripAndSelectName(foundContainer.Names)
	// TODO: FIXME: Use uniform container state representation and convert to it
	canonicalizedContainerState := strings.ToUpper(foundContainer.State)
	if canonicalizedContainerState != "RUNNING" {
		validationErr := taskerrors.NewContainerStateAbnormalError(canonicalizedContainerState)
		return validationErr.Param(), validationErr
	}

	return "", nil
}

func (p *DockerProcessor) Prepare(commandContent string) error {
	p.CommandContent = commandContent
	return nil
}

func (p *DockerProcessor) SyncRun(
	stdoutWriter io.Writer,
	stderrWriter io.Writer,
	stdinReader io.Reader) (int, int, error) {
	// 1. Create an exec instance
	compiledCommand := []string{"/bin/sh", "-c", p.CommandContent}
	execConfig := types.ExecConfig{
		User:         p.Username,
		Tty:          false,
		AttachStdin:  stdinReader != nil,
		AttachStderr: stderrWriter != nil,
		AttachStdout: stdoutWriter != nil,
		WorkingDir:   p.WorkingDirectory,
		Cmd:          compiledCommand,
	}
	execution, err := createExec(p.client, p.foundContainers[0].ID, execConfig, defaultTimeout)
	if err != nil {
		return 1, process.Fail, taskerrors.NewContainerRuntimeInternalError(err)
	}

	// 2. Start the created exec instance and get a hijacked response stream
	execStartConfig := types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	}
	hijackedResponse, err := startAndAttachExec(p.client, execution.ID, execStartConfig, defaultTimeout)
	if err != nil {
		return 1, process.Fail, taskerrors.NewContainerRuntimeInternalError(err)
	}
	defer hijackedResponse.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.Timeout)*time.Second)
	defer cancel()
	p.cancel = cancel

	// Run streamer for stdin/stdout/stderr on hijacked connection concurrently,
	// and catch session termination through channel
	streamed := make(chan error, 1)
	go func() {
		streamed <- streamHijacked(ctx, hijackedResponse, stdoutWriter, stderrWriter, stdinReader)
	}()

	// Wait for command finished, or timeout
	select {
	case <-ctx.Done():
		if ctxErr := ctx.Err(); ctxErr != nil {
			if ctxErr == context.DeadlineExceeded {
				return 1, process.Timeout, errors.New("timeout")
			}
		}
	case err := <-streamed:
		if err != nil {
			return 1, process.Fail, taskerrors.NewContainerRuntimeInternalError(err)
		}
	}

	// Determine process state after exec session terminated.
	// As https://github.com/Mirantis/cri-dockerd/blob/17229a014b98b47966f98a16d4dd9faa5230a31f/core/exec.go#L153-L154
	// says, try to inspect an exec session a few times for the newest state.
	var finalErr error
	var exitCode int
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for retries := 0; ; {
		inspection, err := inspectExec(p.client, execution.ID, defaultTimeout)
		if err != nil {
			finalErr = err
			break
		}

		if !inspection.Running {
			exitCode = inspection.ExitCode
			break
		}

		retries++
		if retries == maxInspectionRetries {
			log.GetLogger().WithFields(logrus.Fields{
				"containerId":   p.ContainerId,
				"containerName": p.ContainerName,
				"execId":        execution.ID,
			}).WithError(ErrInconsistentExecProcessState).Errorln("Failed to conclude process state after exec session")
			return 1, process.Fail, taskerrors.NewContainerRuntimeInternalError(ErrInconsistentExecProcessState)
		}

		<-ticker.C
	}
	if finalErr != nil {
		return 1, process.Fail, taskerrors.NewContainerRuntimeInternalError(finalErr)
	}

	return exitCode, process.Success, nil
}

func (p *DockerProcessor) Cancel() {
	return
}

func (p *DockerProcessor) Cleanup(removeScriptFile bool) error {
	return nil
}

func (p *DockerProcessor) SideEffect() error {
	return nil
}

func (p *DockerProcessor) ExtraLubanParams() string {
	if len(p.foundContainers) != 1 {
		return fmt.Sprintf("&containerId=%s&containerName=%s", p.ContainerId, p.ContainerName)
	}

	return fmt.Sprintf("&containerId=%s&containerName=%s", p.foundContainers[0].ID, p.foundContainerName)
}
