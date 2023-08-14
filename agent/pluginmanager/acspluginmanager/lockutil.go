package acspluginmanager

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aliyun/aliyun_assist_client/common/filelock"
)

func openPluginLockFile(pluginName string) (*os.File, error) {
	pluginManagerCachePath, err := getPluginManagerCachePath()
	if err != nil {
		return nil, err
	}

	pluginLockPath := filepath.Join(pluginManagerCachePath, pluginName + ".lock")
	return os.OpenFile(pluginLockPath, os.O_RDONLY | os.O_CREATE, os.FileMode(0o640))
}

func openPluginVersionLockFile(pluginName string, version string) (*os.File, error) {
	pluginManagerCachePath, err := getPluginManagerCachePath()
	if err != nil {
		return nil, err
	}

	pluginVersionLockPath := filepath.Join(pluginManagerCachePath, fmt.Sprintf("%s.v%s.lock", pluginName, version))
	return os.OpenFile(pluginVersionLockPath, os.O_RDONLY | os.O_CREATE, os.FileMode(0o640))
}

type InstallLockGuard struct {
	PluginName string
	PluginVersion string

	AcquireSharedLockTimeout time.Duration
}

func (g InstallLockGuard) Do(onExclusiveLocked func(), onSharedLocked func()) error {
	pluginVersionLockFile, err := openPluginVersionLockFile(g.PluginName, g.PluginVersion)
	if err != nil {
		return NewOpenPluginVersionLockFileError(err)
	}
	defer pluginVersionLockFile.Close()

	if err = filelock.TryLock(pluginVersionLockFile); err == nil {
		defer filelock.Unlock(pluginVersionLockFile)
		onExclusiveLocked()

		return nil
	} else if !filelock.IsLockingError(err) {
		return NewAcquirePluginVersionExclusiveLockError(err)
	}

	if err := filelock.RLock(pluginVersionLockFile, g.AcquireSharedLockTimeout); err != nil {
		return NewAcquirePluginVersionSharedLockError(err)
	}
	defer filelock.Unlock(pluginVersionLockFile)
	onSharedLocked()
	return nil
}
