package pluginmanager

import (
	"errors"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/pluginmodel"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

func TestShimManagerFindInstalled(t *testing.T) {
	t.Run("OnError", func(t *testing.T) {
		guard := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return nil, os.ErrNotExist
		})
		defer guard.Reset()

		installeds, err := (&ShimManager{}).FindInstalled(nil)
		assert.Nil(t, installeds)
		assert.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("NoInstalled", func(t *testing.T) {
		guard := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{}, nil
		})
		defer guard.Reset()

		installeds, err := (&ShimManager{}).FindInstalled(nil)
		assert.Equal(t, 0, len(installeds))
		assert.Nil(t, err)
	})

	t.Run("TwoInstalled", func(t *testing.T) {
		guard := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{
				{ Name: "Foo", OSType: "Linux" },
				{ Name: "Bar", Arch: "x86_64" },
			}, nil
		})
		defer guard.Reset()

		installeds, err := (&ShimManager{}).FindInstalled(nil)
		assert.Equal(t, 2, len(installeds))
		assert.Equal(t, "Foo", installeds[0].Name())
		assert.Equal(t, "Linux", installeds[0].OSType())
		assert.Equal(t, "Bar", installeds[1].Name())
		assert.Equal(t, "x86_64", installeds[1].Architecture())
		assert.Nil(t, err)
	})

	t.Run("ThreeInstalledButOneRemoved", func(t *testing.T) {
		guard := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{
				{ Name: "Foo", OSType: "Linux" },
				{ Name: "Bar", Arch: "x86_64", IsRemoved: true},
				{ Name: "Baz", Arch: "i686" },
			}, nil
		})
		defer guard.Reset()

		installeds, err := (&ShimManager{}).FindInstalled(nil)
		assert.Equal(t, 2, len(installeds))
		assert.Equal(t, "Foo", installeds[0].Name())
		assert.Equal(t, "Linux", installeds[0].OSType())
		assert.Equal(t, "Baz", installeds[1].Name())
		assert.Equal(t, "i686", installeds[1].Architecture())
		assert.Nil(t, err)
	})
}

func TestShimManagerHealthCheck(t *testing.T) {
	t.Run("SuccessfulHealthCheckByScan", func(t *testing.T) {
		sm := &ShimManager{}
		mockReturn := []pluginmodel.PluginStatus{
			{Name: "test", Status: pluginmodel.PERSIST_RUNNING},
		}
		guard := gomonkey.ApplyPrivateMethod(
			reflect.TypeOf(sm), "healthCheckByScan",
			func (*ShimManager, logrus.FieldLogger) ([]pluginmodel.PluginStatus, error) {
				return mockReturn, nil
			},
		)
		defer guard.Reset()

		statuses, err := sm.HealthCheck(nil, pluginmodel.HealthCheckByScan)
		assert.NoError(t, err)
		assert.Equal(t, mockReturn, statuses)
	})

	t.Run("FailedHealthCheckByScan", func(t *testing.T) {
		sm := &ShimManager{}
		guard := gomonkey.ApplyPrivateMethod(
			reflect.TypeOf(sm), "healthCheckByScan",
			func (*ShimManager, logrus.FieldLogger) ([]pluginmodel.PluginStatus, error) {
				return nil, errors.New("mock error")
			},
		)
		defer guard.Reset()

		noStatuses, err := sm.HealthCheck(nil, pluginmodel.HealthCheckByScan)
		assert.Nil(t, noStatuses)
		assert.Error(t, err)
	})

	t.Run("SuccessfulHealthCheckByPull", func(t *testing.T) {
		sm := &ShimManager{}
		mockReturn := []pluginmodel.PluginStatus{
			{Name: "test", Status: pluginmodel.ONCE_INSTALLED},
		}
		guard := gomonkey.ApplyPrivateMethod(
			reflect.TypeOf(sm), "healthCheckByPull",
			func (*ShimManager, logrus.FieldLogger) ([]pluginmodel.PluginStatus, error) {
				return mockReturn, nil
			},
		)
		defer guard.Reset()

		statuses, err := sm.HealthCheck(nil, pluginmodel.HealthCheckByPull)
		assert.NoError(t, err)
		assert.Equal(t, mockReturn, statuses)
	})

	t.Run("FailedHealthCheckByPull", func(t *testing.T) {
		sm := &ShimManager{}
		guard := gomonkey.ApplyPrivateMethod(
			reflect.TypeOf(sm), "healthCheckByPull",
			func (*ShimManager, logrus.FieldLogger) ([]pluginmodel.PluginStatus, error) {
				return nil, errors.New("mock error")
			},
		)
		defer guard.Reset()

		noStatuses, err := sm.HealthCheck(nil, pluginmodel.HealthCheckByPull)
		assert.Nil(t, noStatuses)
		assert.Error(t, err)
	})

	t.Run("InvalidStrategy", func(t *testing.T) {
		sm := &ShimManager{}
		noStatuses, err := sm.HealthCheck(nil, pluginmodel.HealthCheckStrategy("InvalidStrategy"))
		assert.Nil(t, noStatuses)
		assert.NoError(t, err)
	})
}

func TestHealthCheckByScan(t *testing.T) {
	noLogger := logrus.New()
	noLogger.Out = io.Discard

	t.Run("FindInstalledError", func(t *testing.T) {
		nestedErr := errors.New("find error")
		guard := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return nil, nestedErr
		})
		defer guard.Reset()

		_, err := (&ShimManager{}).healthCheckByScan(nil)
		assert.ErrorContains(t, err, "failed to load installed plugins")
		assert.ErrorIs(t, err, nestedErr)
	})

	t.Run("NoPlugins", func(t *testing.T) {
        findMock := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
            return []PluginInfo{}, nil
        })
        defer findMock.Reset()

        sm := &ShimManager{}
        statuses, err := sm.healthCheckByScan(noLogger)
        assert.NoError(t, err)
        assert.Empty(t, statuses)
    })

    t.Run("OnlyLongRunningPlugin", func(t *testing.T) {
        mockPlugins := []PluginInfo{
            {
				Name:        "test-persist",
				Version:     "1.0",
				PluginType_: PLUGIN_PERSIST,
				IsRemoved:   false},
        }
        findMock := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
            return mockPlugins, nil
        })
        defer findMock.Reset()

        mockOutput := `[{"Name":"test-persist","Version":"1.0","Status":"PERSIST_RUNNING"}]`
        runMock := gomonkey.ApplyFunc(syncRunKillGroup, func(workingDir string, cmd string, args []string, stdout, stderr io.Writer, timeout int) (int, int, error) {
            if cmd == "acs-plugin-manager" && args[0] == "--status" {
                stdout.Write([]byte(mockOutput))
                return 0, 0, nil
            }
            return 0, 0, nil
        })
        defer runMock.Reset()

        sm := &ShimManager{}
        statuses, err := sm.healthCheckByScan(noLogger)
        assert.NoError(t, err)
        assert.Len(t, statuses, 1)
        assert.Equal(t, pluginmodel.PERSIST_RUNNING, statuses[0].Status)
    })

	t.Run("MixedPlugins", func(t *testing.T) {
        mockPlugins := []PluginInfo{
            {Name: "once-plugin", Version: "1.0", PluginType_: PLUGIN_ONCE, IsRemoved: false},
            {Name: "persist-ok", Version: "2.0", PluginType_: PLUGIN_PERSIST, IsRemoved: false},
        }
        findMock := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
            return mockPlugins, nil
        })
        defer findMock.Reset()

        mockOutput := `[{"Name":"persist-ok","Version":"2.0","Status":"PERSIST_RUNNING"}]`
        runMock := gomonkey.ApplyFunc(syncRunKillGroup, func(workingDir string, cmd string, args []string, stdout, stderr io.Writer, timeout int) (int, int, error) {
            if args[0] == "--status" {
                stdout.Write([]byte(mockOutput))
                return 0, 0, nil
            }
            return 0, 0, nil
        })
        defer runMock.Reset()

        sm := &ShimManager{}
        statuses, err := sm.healthCheckByScan(noLogger)
        assert.NoError(t, err)
        assert.Len(t, statuses, 2)
        assert.Equal(t, pluginmodel.ONCE_INSTALLED, statuses[0].Status)
        assert.Equal(t, pluginmodel.PERSIST_RUNNING, statuses[1].Status)
    })

    t.Run("CommandError", func(t *testing.T) {
        mockPlugins := []PluginInfo{{Name: "test-plugin", PluginType_: PLUGIN_PERSIST}}
        findMock := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
            return mockPlugins, nil
        })
        defer findMock.Reset()

        runMock := gomonkey.ApplyFunc(syncRunKillGroup, func(workingDir string, cmd string, args []string, stdout, stderr io.Writer, timeout int) (int, int, error) {
            return 1, 1, errors.New("command failed")
        })
        defer runMock.Reset()

        sm := &ShimManager{}
        _, err := sm.healthCheckByScan(noLogger)
        assert.Error(t, err)
    })

    t.Run("PersistPluginFail", func(t *testing.T) {
        mockPlugins := []PluginInfo{
            {
                Name:        "failed-plugin",
                Version:     "1.0",
                PluginType_:  PLUGIN_PERSIST,
                IsRemoved:   false,
                Timeout:     "60",
            },
        }
        findMock := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
            return mockPlugins, nil
        })
        defer findMock.Reset()

        mockOutput := `[{"Name":"failed-plugin","Version":"1.0","Status":"PERSIST_FAIL"}]`
        runMock := gomonkey.ApplyFunc(syncRunKillGroup, func(workingDir string, cmd string, args []string, stdout, stderr io.Writer, timeout int) (int, int, error) {
            if args[0] == "--status" {
                stdout.Write([]byte(mockOutput))
                return 0, 0, nil
            }
            if args[len(args)-1] == "--start" {
                return 0, 0, nil
            }
            return 0, 0, nil
        })
        defer runMock.Reset()

        // 禁用随机延迟
        randMock := gomonkey.ApplyFunc(rand.Intn, func(n int) int { return 0 })
        timeMock := gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {})
        defer randMock.Reset()
        defer timeMock.Reset()

        sm := &ShimManager{}
        statuses, err := sm.healthCheckByScan(noLogger)
        assert.NoError(t, err)
        assert.Len(t, statuses, 0)
    })
}

func TestShimManagerHealthCheckByPull(t *testing.T) {
	noLogger := logrus.New()
	noLogger.Out = io.Discard

	t.Run("FindInstalledError", func(t *testing.T) {
		nestedErr := errors.New("find error")
		guard := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return nil, nestedErr
		})
		defer guard.Reset()

		_, err := (&ShimManager{}).healthCheckByPull(nil)
		assert.ErrorContains(t, err, "failed to load installed plugins")
		assert.ErrorIs(t, err, nestedErr)
	})

	t.Run("FindInstalledPathError", func(t *testing.T) {
		nestedErr := errors.New("path error")
		guard := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return "", nestedErr
		})
		defer guard.Reset()

		_, err := (&ShimManager{}).healthCheckByPull(nil)
		assert.ErrorContains(t, err, "failed to load installed plugins")
		assert.ErrorIs(t, err, nestedErr)
	})

	t.Run("NoPlugins", func(t *testing.T) {
        findMock := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
            return []PluginInfo{}, nil
        })
        defer findMock.Reset()

        sm := &ShimManager{}
        statuses, err := sm.healthCheckByPull(nil)
        assert.NoError(t, err)
        assert.Empty(t, statuses)
    })

	t.Run("LateGetPluginPathError", func(t *testing.T) {
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return make([]PluginInfo, 1), nil
		})
		defer guard1.Reset()

		nestedErr := errors.New("path error")
		guard2 := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return "", nestedErr
		})
		defer guard2.Reset()

		_, err := (&ShimManager{}).healthCheckByPull(nil)
		assert.ErrorContains(t, err, "failed to resolve global plugin directory")
		assert.ErrorIs(t, err, nestedErr)
	})

	t.Run("SuccessWithValidHeartbeat", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "valid-plugin",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_PERSIST,
			HeartbeatInterval:  10,
			IsRemoved:          false,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		pluginDir := "/mock/plugin/path"
		guard2 := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return pluginDir, nil
		})
		defer guard2.Reset()

		heartbeatPath := filepath.Join(pluginDir, mockPlugin.Name, mockPlugin.Version, "heartbeat")
		guard3 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(path string) bool {
			return path == heartbeatPath
		})
		defer guard3.Reset()

		now := time.Now().Unix()
		validTimestamp := strconv.FormatInt(now-5, 10)
		guard4 := gomonkey.ApplyFunc(os.ReadFile, func(path string) ([]byte, error) {
			if path == heartbeatPath {
				return []byte(validTimestamp), nil
			}
			return nil, errors.New("unexpected path")
		})
		defer guard4.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(noLogger)
		assert.NoError(t, err)
		assert.Len(t, statuses, 1)
		assert.Equal(t, pluginmodel.PERSIST_RUNNING, statuses[0].Status)
	})

	t.Run("HeartbeatTimeout", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "timeout-plugin",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_PERSIST,
			HeartbeatInterval:  10,
			IsRemoved:          false,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		pluginDir := "/mock/plugin/path"
		guard2 := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return pluginDir, nil
		})
		defer guard2.Reset()

		heartbeatPath := filepath.Join(pluginDir, mockPlugin.Name, mockPlugin.Version, "heartbeat")
		guard3 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(path string) bool {
			return path == heartbeatPath
		})
		defer guard3.Reset()

		now := time.Now().Unix()
		invalidTimestamp := strconv.FormatInt(now-20, 10)
		guard4 := gomonkey.ApplyFunc(os.ReadFile, func(path string) ([]byte, error) {
			if path == heartbeatPath {
				return []byte(invalidTimestamp), nil
			}
			return nil, errors.New("unexpected path")
		})
		defer guard4.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(noLogger)
		assert.NoError(t, err)
		assert.Len(t, statuses, 1)
		assert.Equal(t, pluginmodel.PERSIST_FAIL, statuses[0].Status)
	})

	t.Run("HeartbeatFileNotFound", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "no-heartbeat",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_PERSIST,
			HeartbeatInterval:  10,
			IsRemoved:          false,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		pluginDir := "/mock/plugin/path"
		guard2 := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return pluginDir, nil
		})
		defer guard2.Reset()

		heartbeatPath := filepath.Join(pluginDir, mockPlugin.Name, mockPlugin.Version, "heartbeat")
		guard3 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(path string) bool {
			// Return false as specified file not found
			return path != heartbeatPath
		})
		defer guard3.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(noLogger)
		// No heartbeat file is acceptable and would not be taken into account
		assert.NoError(t, err)
		assert.Len(t, statuses, 0)
	})

	t.Run("ReadHeartbeatError", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "read-error",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_PERSIST,
			HeartbeatInterval:  10,
			IsRemoved:          false,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		pluginDir := "/mock/plugin/path"
		guard2 := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return pluginDir, nil
		})
		defer guard2.Reset()

		heartbeatPath := filepath.Join(pluginDir, mockPlugin.Name, mockPlugin.Version, "heartbeat")
		guard3 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(path string) bool {
			return path == heartbeatPath
		})
		defer guard3.Reset()

		guard4 := gomonkey.ApplyFunc(os.ReadFile, func(path string) ([]byte, error) {
			return nil, errors.New("read error")
		})
		defer guard4.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(noLogger)
		// Error encountered during reading heart-beat file does not count
		assert.NoError(t, err)
		assert.Len(t, statuses, 0)
	})

	t.Run("InvalidTimestampFormat", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "invalid-timestamp",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_PERSIST,
			HeartbeatInterval:  10,
			IsRemoved:          false,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		pluginDir := "/mock/plugin/path"
		guard2 := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
			return pluginDir, nil
		})
		defer guard2.Reset()

		heartbeatPath := filepath.Join(pluginDir, mockPlugin.Name, mockPlugin.Version, "heartbeat")
		guard3 := gomonkey.ApplyFunc(fileutil.CheckFileIsExist, func(path string) bool {
			return path == heartbeatPath
		})
		defer guard3.Reset()

		guard4 := gomonkey.ApplyFunc(os.ReadFile, func(path string) ([]byte, error) {
			return []byte("invalid"), nil
		})
		defer guard4.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(noLogger)
		assert.NoError(t, err)
		assert.Len(t, statuses, 0)
	})

	t.Run("NonPersistentPluginIgnored", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "non-persist",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_ONCE,
			HeartbeatInterval:  10,
			IsRemoved:          false,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(nil)
		// One-shot plugin needs no heart-beat measuring
		assert.NoError(t, err)
		assert.Len(t, statuses, 0)
	})

	t.Run("RemovedPluginIgnored", func(t *testing.T) {
		mockPlugin := PluginInfo{
			Name:               "removed-plugin",
			Version:            "1.0.0",
			PluginType_:        PLUGIN_PERSIST,
			HeartbeatInterval:  10,
			IsRemoved:          true,
		}
		guard1 := gomonkey.ApplyFunc(_findAllInstalledPlugins, func() ([]PluginInfo, error) {
			return []PluginInfo{mockPlugin}, nil
		})
		defer guard1.Reset()

		statuses, err := (&ShimManager{}).healthCheckByPull(nil)
		// Plugin has been removed does not matter
		assert.NoError(t, err)
		assert.Len(t, statuses, 0)
	})
}
