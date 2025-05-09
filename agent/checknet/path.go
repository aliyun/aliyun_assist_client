package checknet

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/util/versionutil"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

const (
	defaultCommanderName = "ACS-ECS-SysInfoGatherer"
)

var (
	_netcheckPath     string
	_netcheckPathLock sync.Mutex
)

// initNetcheckPath detects whether netcheck program is bundled within current
// agent release version
func initNetcheckPath() error {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"module": "checknet",
	})

	pluginDir, err := pathutil.GetPluginPath()
	if err != nil {
		_netcheckPath = ""
		return err
	}
	preInstalledPluginDir, err := pathutil.GetPreInstalledPluginPath()
	if err != nil {
		_netcheckPath = ""
		return err
	}

	var currentVersionNetcheckPath string
	installedPlugin, err1 := acspluginmanager.QueryPluginFromLocal(defaultCommanderName, pluginmanager.PLUGIN_COMMANDER)
	preInstalledPlugin, err2 := acspluginmanager.QueryPluginFromLocalPreInstalled(defaultCommanderName, pluginmanager.PLUGIN_COMMANDER)
	if err1 != nil && err2 != nil {
		logger.Errorf("query installed plugin failed:%v; query pre-installed plugin failed:%v", err1, err2)
		return fmt.Errorf("query installed plugin failed:%v; query pre-installed plugin failed:%v", err1, err2)
	} else if err1 != nil {
		logger.WithError(err1).Errorln("Failed to query installed plugin, use pre-installed plugin")
		currentVersionNetcheckPath = filepath.Join(preInstalledPluginDir, defaultCommanderName, preInstalledPlugin.Version, preInstalledPlugin.RunPath)
	} else if err2 != nil {
		logger.WithError(err2).Errorln("Failed to query pre-installed plugin, use installed plugin")
		currentVersionNetcheckPath = filepath.Join(pluginDir, defaultCommanderName, installedPlugin.Version, installedPlugin.RunPath)
	} else {
		// compare version
		if versionutil.CompareVersion(installedPlugin.Version, preInstalledPlugin.Version) > 0 {
			logger.Infoln("Use installed plugin")
			currentVersionNetcheckPath = filepath.Join(pluginDir, defaultCommanderName, installedPlugin.Version, installedPlugin.RunPath)
		} else {
			logger.Infoln("Use pre-installed plugin")
			currentVersionNetcheckPath = filepath.Join(preInstalledPluginDir, defaultCommanderName, preInstalledPlugin.Version, preInstalledPlugin.RunPath)
		}
	}

	if !fileutil.CheckFileIsExist(currentVersionNetcheckPath) {
		_netcheckPath = ""
		logger.Errorf("Netcheck executable not found at %s", currentVersionNetcheckPath)
		return fmt.Errorf("Netcheck executable not found at %s", currentVersionNetcheckPath)
	}
	if !fileutil.CheckFileIsExecutable(currentVersionNetcheckPath) {
		os.Chmod(currentVersionNetcheckPath, os.FileMode(0o744))
	}

	_netcheckPath = currentVersionNetcheckPath
	return nil
}

func getNetcheckPath() string {
	_netcheckPathLock.Lock()
	defer _netcheckPathLock.Unlock()
	if _netcheckPath == "" || !fileutil.CheckFileIsExist(_netcheckPath) {
		if err := initNetcheckPath(); err != nil {
			log.GetLogger().WithFields(logrus.Fields{
				"module": "checknet",
			}).WithError(err).Errorln("Failed to detect netcheck executable path")
		}
	}

	return _netcheckPath
}
