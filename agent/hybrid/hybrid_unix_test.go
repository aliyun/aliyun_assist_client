//go:build !windows
// +build !windows

package hybrid

import (
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util/serviceutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestCleanUpRegisterDataAndExit(t *testing.T) {
	var (
		restartAgentServiceCalled bool
		exitCalled                bool
		exitCode                  int
	)

	defer gomonkey.ApplyFunc(serviceutil.RestartAgentService, func(logrus.FieldLogger) {
		restartAgentServiceCalled = true
	}).Reset()
	defer gomonkey.ApplyFunc(os.Exit, func(code int) {
		exitCode = code
		exitCalled = true
	}).Reset()

	CleanUpRegisterDataAndExit()
	assert.False(t, restartAgentServiceCalled)
	assert.True(t, exitCalled)
	assert.Equal(t, 0, exitCode)
}
