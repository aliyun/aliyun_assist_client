package cri

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"k8s.io/kubernetes/pkg/probe/exec"
	utilexec "k8s.io/utils/exec"

	libcri "github.com/aliyun/aliyun_assist_client/agent/container/cri"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
)

type CRIProcessor struct {
	TaskId string
	// Fundamental properties of command process
	ContainerIdentifier string
	ContainerName string
	CommandType string
	CommandContent string
	Timeout int

	// Extracted properties about target container
	runtimeEndpoints []libcri.RuntimeEndpoint
	containerId string

	// Connection to container runtime service
	connection *containerConnection
}

func (p *CRIProcessor) PreCheck() (string, error) {
	identifierSegments := strings.Split(p.ContainerIdentifier, "://")
	if len(identifierSegments) == 1 {
		p.runtimeEndpoints = nil
		p.containerId = p.ContainerIdentifier
		return "", nil
	}

	// Or the type of CRI-compatible container runtime has been extracted from
	// container identifier in format <type>://<container-id>
	if identifierSegments[1] == "" {
		validationErr := taskerrors.NewInvalidContainerIdError()
		return validationErr.Param(), validationErr
	}
	supportedEndpoints, ok := libcri.RuntimeName2Endpoints[identifierSegments[0]]
	if !ok {
		validationErr := taskerrors.NewUnsupportedContainerRuntimeError()
		return validationErr.Param(), validationErr
	}

	p.runtimeEndpoints = supportedEndpoints
	p.containerId = identifierSegments[1]
	return "", nil
}

func (p *CRIProcessor) Prepare(commandContent string) error {
	var err error
	p.connection, err = getRuntimeService(p.runtimeEndpoints, 10 * time.Second, p.containerId, p.ContainerName)
	if err != nil {
		return err
	}

	p.CommandContent = commandContent
	return nil
}

func (p *CRIProcessor) SyncRun(
		stdoutWriter io.Writer,
		stderrWriter io.Writer,
		stdinReader  io.Reader)  (exitCode int, status int, err error) {
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

func (p *CRIProcessor) Cancel() {
	return
}

func (p *CRIProcessor) Cleanup(removeScriptFile bool) error {
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
