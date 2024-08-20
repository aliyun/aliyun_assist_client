package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"time"

	dockerutil "github.com/aliyun/aliyun_assist_client/plugin/commander/container/util/docker"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/docker/docker/api/types"
	dockerclient "github.com/docker/docker/client"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
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
	SubmissionID string
	// Fundamental properties of command process
	ContainerId    string
	ContainerName  string // may be filled when only container id specified
	CommandType    string
	CommandContent string
	Timeout        int
	// Additional execution attributes supported by docker
	WorkingDirectory string
	Username         string

	client    *dockerclient.Client
	container *types.Container
	// stripped and selected name for the container found
	foundContainerName string
	cancel             context.CancelFunc
}

func NewProcessor(options *model.ContainerCommandOptions, client *dockerclient.Client, container *types.Container) *DockerProcessor {
	return &DockerProcessor{
		SubmissionID: options.SubmissionID,
		// Fundamental properties of command process
		ContainerId:        options.SubmissionID,
		ContainerName:      options.ContainerName,
		CommandType:        options.CommandType,
		Timeout:            options.Timeout,
		WorkingDirectory:   options.WorkingDirectory,
		Username:           options.Username,
		client:             client,
		container:          container,
		foundContainerName: dockerutil.StripAndSelectName(container.Names),
	}
}

// PreCheck method of DockerProcessor struct continues to perform pre-checking
// actions left by CheckDockerProcessor() function.
func (p *DockerProcessor) PreCheck() (string, *taskerrors.TaskError) {
	return "", nil
}

func (p *DockerProcessor) Prepare(commandContent string) *taskerrors.TaskError {
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
	execution, err := createExec(p.client, p.container.ID, execConfig, defaultTimeout)
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

func (p *DockerProcessor) Cancel() error {
	return nil
}

func (p *DockerProcessor) Cleanup() error {
	return nil
}

func (p *DockerProcessor) SideEffect() error {
	p.client.Close()
	return nil
}

func (p *DockerProcessor) ExtraLubanParams() string {
	return fmt.Sprintf("&containerId=%s&containerName=%s", p.container.ID, p.foundContainerName)
}
