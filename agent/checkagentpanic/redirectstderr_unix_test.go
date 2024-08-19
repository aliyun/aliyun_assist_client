//go:build !windows

package checkagentpanic

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/stretchr/testify/assert"
)

var (
	_checkPointLog       string
	_panicLog            string
	_checkPointTimestamp int64 // second
	_panicTimestamp      int64
)

func TestRedirectStdouterr(t *testing.T) {
	stdouterrDir, _ := pathutil.GetLogPath()
	stdoutFile := filepath.Join(stdouterrDir, stdoutFileName)
	stderrFile := filepath.Join(stdouterrDir, stderrFileName)
	defer func() {
		os.RemoveAll(stdouterrDir)
	}()

	guard_isSystemd := gomonkey.ApplyFunc(util.IsSystemdLinux, func() bool {
		return false
	})
	defer guard_isSystemd.Reset()

	fmt.Fprintf(os.Stdout, "stdout: should not record")
	fmt.Fprintf(os.Stderr, "stderr: should not record")

	RedirectStdouterr()

	stdoutContent := `stdout: should record 1
stdout: should record 2`
	stderrContent := `stderr: should record 1
stderr: should record 2`
	fmt.Fprint(os.Stdout, stdoutContent)
	fmt.Fprint(os.Stderr, stderrContent)

	stdoutFileContent, _ := os.ReadFile(stdoutFile)
	stderrFileContent, _ := os.ReadFile(stderrFile)
	assert.Equal(t, stdoutContent, string(stdoutFileContent))
	assert.Equal(t, stderrContent, string(stderrFileContent))
}
