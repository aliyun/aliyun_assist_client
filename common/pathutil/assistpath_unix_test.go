//go:build !windows
// +build !windows

package pathutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestEnvSet(t *testing.T) {
	os.Setenv("path1", "test")
	path := os.Getenv("path1")
	assert.Equal(t, path, "test")
}

func TestGetUnixXxxPath(t *testing.T) {
	guard_1 := gomonkey.ApplyFunc(os.Executable, func() (string, error) {
		return "/usr/local/share/aliyun-assist/2.2.3.999/aliyun-service", nil
	})
	defer guard_1.Reset()
	scriptPath, _ := GetScriptPath()
	hybridPath, _ := GetHybridPath(GPNoCreation)
	configPath, _ := GetConfigPath()
	crossVersionConfigPath, _ := GetCrossVersionConfigPath()
	cachePath, _ := GetCachePath()
	pluginPath, _ := GetPluginPath()
	assert.Equal(t, "/usr/local/share/aliyun-assist/work/script", scriptPath)
	assert.Equal(t, "/usr/local/share/aliyun-assist/hybrid", hybridPath)
	assert.Equal(t, "/usr/local/share/aliyun-assist/2.2.3.999/config", configPath)
	assert.Equal(t, "/usr/local/share/aliyun-assist/config", crossVersionConfigPath)
	assert.Equal(t, "/usr/local/share/aliyun-assist/cache", cachePath)
	assert.Equal(t, "/usr/local/share/aliyun-assist/plugin", pluginPath)
}

func TestGetHybridPath(t *testing.T) {
	t.Run("GetCrossVersionInboundDirFailed", func(t *testing.T) {
		// Mock GetCrossVersionInboundDir to return error
		patches := gomonkey.ApplyFunc(GetCrossVersionInboundDir, func() (string, error) {
			return "", errors.New("mocked error")
		})
		defer patches.Reset()

		path, err := GetHybridPath()
		assert.Error(t, err, "Expected error when GetCrossVersionInboundDir fails")
		assert.Empty(t, path, "Path should be empty on error")
	})

	const mockCrossVersionInboundDir = "/mock/cross/version/dir"
	expectedPath := filepath.Join(mockCrossVersionInboundDir, "hybrid")
	t.Run("MakeSurePathFailed", func(t *testing.T) {
		// Mock GetCrossVersionInboundDir
		patches := gomonkey.ApplyFunc(GetCrossVersionInboundDir, func() (string, error) {
			return mockCrossVersionInboundDir, nil
		})
		// Mock MakeSurePath failure
		mockErr := errors.New("mkdir failed")
		patches.ApplyFunc(MakeSurePath, func(path string) error {
			assert.Equal(t, expectedPath, path, "MakeSurePath called with unexpected path")
			return mockErr
		})
		defer patches.Reset()

		path, err := GetHybridPath()
		assert.Error(t, err, "Expected error when MakeSurePath fails")
		assert.Equal(t, mockErr, err, "Error mismatch when MakeSurePath fails")
		assert.Equal(t, expectedPath, path, "Path mismatch when MakeSurePath fails")
	})

	t.Run("ShouldNotCreatePathWithGPNoCreationSet", func(t *testing.T) {
		// Mock GetCrossVersionInboundDir success
		patches := gomonkey.ApplyFunc(GetCrossVersionInboundDir, func() (string, error) {
			return mockCrossVersionInboundDir, nil
		})
		// Mock MakeSurePath to verify it's not called
		called := false
		patches.ApplyFunc(MakeSurePath, func(path string) error {
			called = true
			return nil
		})
		defer patches.Reset()

		path, err := GetHybridPath(GPNoCreation)
		assert.NoError(t, err, "Unexpected error when GPNoCreation is set")
		assert.Equal(t, expectedPath, path, "Path mismatch with GPNoCreation")
		assert.False(t, called, "MakeSurePath should not be called when GPNoCreation is set")
	})

	t.Run("NormalCaseWithoutGPNoCreationSet", func(t *testing.T) {
		// Mock GetCrossVersionInboundDir
		patches := gomonkey.ApplyFunc(GetCrossVersionInboundDir, func() (string, error) {
			return mockCrossVersionInboundDir, nil
		})
		// Mock MakeSurePath success
		called := false
		patches.ApplyFunc(MakeSurePath, func(path string) error {
			assert.Equal(t, expectedPath, path, "MakeSurePath called with unexpected path")
			called = true
			return nil
		})
		defer patches.Reset()

		path, err := GetHybridPath()
		assert.NoError(t, err, "Unexpected error in default case")
		assert.Equal(t, expectedPath, path, "Path mismatch in default case")
		assert.True(t, called, "MakeSurePath should be called in default case")
	})

	t.Run("OnlyZeroValueOptionPassed", func(t *testing.T) {
		// Mock GetCrossVersionInboundDir
		patches := gomonkey.ApplyFunc(GetCrossVersionInboundDir, func() (string, error) {
			return mockCrossVersionInboundDir, nil
		})
		// Mock MakeSurePath success
		called := false
		patches.ApplyFunc(MakeSurePath, func(path string) error {
			assert.Equal(t, expectedPath, path, "MakeSurePath called with unexpected path")
			called = true
			return nil
		})
		defer patches.Reset()

		path, err := GetHybridPath(0)
		assert.NoError(t, err, "Unexpected error in default case")
		assert.Equal(t, expectedPath, path, "Path mismatch in default case")
		assert.True(t, called, "MakeSurePath should be called in default case")
	})

	t.Run("OnlyFirstOptionUsedIfMultiplePassed", func(t *testing.T) {
		// Mock GetCrossVersionInboundDir
		guardGet := gomonkey.ApplyFunc(GetCrossVersionInboundDir, func() (string, error) {
			return mockCrossVersionInboundDir, nil
		})
		defer guardGet.Reset()

		// Mock MakeSurePath to verify it's not called
		called := false
		guardMake := gomonkey.ApplyFunc(MakeSurePath, func(path string) error {
			called = true
			return nil
		})
		defer guardMake.Reset()

		// Pass multiple options (GPNoCreation and a dummy one)
		path, err := GetHybridPath(GPNoCreation, 2 /* dummy option */)
		assert.NoError(t, err, "Unexpected error when GPNoCreation is set")
		assert.Equal(t, expectedPath, path, "Path mismatch with multiple options")
		assert.False(t, called, "MakeSurePath should not be called when GPNoCreation is set")
	})
}
