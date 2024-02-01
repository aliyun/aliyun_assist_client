package apiserver

import (
	"bytes"
	"os/exec"
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/common/langutil"
)

var (
	candidateExternalExecutableProviderNames = []string{
		"apiserver-provider.exe",
		"apiserver-provider.bat",
		"apiserver-provider.ps1",
	}
)

func runExternalProvider(executablePath string) (string, string, error) {
	var command *exec.Cmd
	if filepath.Ext(executablePath) == ".ps1" {
		command = exec.Command("powershell.exe", "-file", executablePath)
	} else {
		command = exec.Command(executablePath)
	}

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer

	err := command.Run()
	stdout := langutil.LocalToUTF8(stdoutBuffer.String())
	stderr := langutil.LocalToUTF8(stderrBuffer.String())

	return stdout, stderr, err
}
