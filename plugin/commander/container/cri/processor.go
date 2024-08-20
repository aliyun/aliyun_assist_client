package cri

import (
	"errors"
	"fmt"
	"io"
	"time"

	"k8s.io/kubernetes/pkg/probe/exec"
	utilexec "k8s.io/utils/exec"

	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
)

type CRIProcessor struct {
	SubmissionID string
	// Fundamental properties of command process
	ContainerIdentifier string
	ContainerName       string
	CommandType         string
	CommandContent      string
	Timeout             int

	// Connection to container runtime service
	connection *containerConnection
}

func NewProcessor(options *model.ContainerCommandOptions, connection *containerConnection) *CRIProcessor {
	return &CRIProcessor{
		SubmissionID: options.SubmissionID,
		// Fundamental properties of command process
		ContainerIdentifier: options.ContainerId,
		ContainerName:       options.ContainerName,
		CommandType:         options.CommandType,
		Timeout:             options.Timeout,
		connection:          connection,
	}
}

func (p *CRIProcessor) PreCheck() (string, *taskerrors.TaskError) {
	return "", nil
}

func (p *CRIProcessor) Prepare(commandContent string) *taskerrors.TaskError {
	p.CommandContent = commandContent
	return nil
}

func (p *CRIProcessor) SyncRun(
	stdoutWriter io.Writer,
	stderrWriter io.Writer,
	stdinReader io.Reader) (exitCode int, status int, err error) {
	compiledCommand := []string{"/bin/sh", "-c", p.CommandContent}
	timeout := time.Duration(p.Timeout) * time.Second
	stdout, stderr, err := p.connection.runtimeService.ExecSync(p.connection.containerId, compiledCommand, timeout)

	stdoutWriter.Write(stdout)
	stderrWriter.Write(stderr)

	if err != nil {
		// I know, I know error-handling code below deeps into the CRI client
		// implementation of Kubernetes too much
		var exitcodeErr *utilexec.CodeExitError
		var timeoutErr *exec.TimeoutError
		if errors.As(err, &exitcodeErr) {
			return exitcodeErr.Code, process.Success, nil
		} else if errors.As(err, &timeoutErr) {
			return 1, process.Timeout, err
		} else {
			return 1, process.Fail, taskerrors.NewContainerRuntimeInternalError(err)
		}
	}
	return 0, process.Success, nil
}

func (p *CRIProcessor) Cancel() error {
	return nil
}

func (p *CRIProcessor) Cleanup() error {
	return nil
}

func (p *CRIProcessor) SideEffect() error {
	return nil
}

func (p *CRIProcessor) ExtraLubanParams() string {
	if p.connection == nil {
		return fmt.Sprintf("&containerId=%s&containerName=%s", p.ContainerIdentifier, p.ContainerName)
	}

	return fmt.Sprintf("&containerId=%s&containerName=%s", p.connection.containerId, p.connection.containerName)
}
