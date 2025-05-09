package serviceutil

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/systemdutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestRestartAgentService(t *testing.T) {
	var (
		mockIsSystemd        bool
		mockRestartUnitErr   error
		mockSyncRunSimpleErr error

		RestartUnitCalled      bool
		restartByCommandCalled bool
	)

	defer gomonkey.ApplyFunc(systemdutil.IsRunningSystemd, func() bool {
		return mockIsSystemd
	}).Reset()
	defer gomonkey.ApplyFunc(systemdutil.RestartUnit, func(ctx context.Context, unitName string) (string, error) {
		RestartUnitCalled = true
		return "", mockRestartUnitErr
	}).Reset()
	var p *process.ProcessCmd
	defer gomonkey.ApplyMethod(reflect.TypeOf(p), "SyncRunSimple", func(*process.ProcessCmd, string, []string, int) error {
		restartByCommandCalled = true
		return mockSyncRunSimpleErr
	}).Reset()

	restartBySystemd := func(t *testing.T) {
		mockIsSystemd = true
		mockRestartUnitErr = nil
		RestartUnitCalled = false
		restartByCommandCalled = false

		RestartAgentService(logrus.New())

		assert.True(t, RestartUnitCalled)
		assert.False(t, restartByCommandCalled)
	}

	restartInNoSystemd := func(t *testing.T) {
		mockIsSystemd = false
		mockRestartUnitErr = nil
		RestartUnitCalled = false
		restartByCommandCalled = false

		RestartAgentService(logrus.New())

		assert.False(t, RestartUnitCalled)
		assert.True(t, restartByCommandCalled)
	}

	restartBySystemdFailed := func(t *testing.T) {
		mockIsSystemd = true
		mockRestartUnitErr = errors.New("some err")
		RestartUnitCalled = false
		restartByCommandCalled = false

		RestartAgentService(logrus.New())

		assert.True(t, RestartUnitCalled)
		assert.True(t, restartByCommandCalled)
	}

	restartFailed := func(t *testing.T) {
		mockIsSystemd = true
		mockRestartUnitErr = errors.New("some err")
		mockSyncRunSimpleErr = errors.New("some err")
		RestartUnitCalled = false
		restartByCommandCalled = false

		RestartAgentService(logrus.New())

		assert.True(t, RestartUnitCalled)
		assert.True(t, restartByCommandCalled)
	}

	t.Run("restartBySystemd", restartBySystemd)

	t.Run("restartInNoSystemd", restartInNoSystemd)
	t.Run("restartBySystemdFailed", restartBySystemdFailed)
	t.Run("restartFailed", restartFailed)
}
