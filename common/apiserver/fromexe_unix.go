//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package apiserver

import (
	"bytes"
	"os/exec"
)

var (
	candidateExternalExecutableProviderNames = []string{
		"apiserver-provider",
	}
)

func runExternalProvider(executablePath string) (string, string, error) {
	command := exec.Command(executablePath)

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	err := command.Run()
	return stdoutBuffer.String(), stderrBuffer.String(), err
}
