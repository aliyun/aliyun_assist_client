package pluginmanager

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func getInstalledPluginsBoltPath() (string, error) {
	pluginPath, err := util.GetPluginPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginPath, "installed_plugins.db"), nil
}

func getInstalledPluginsJSONPath() (string, error) {
	pluginPath, err := util.GetPluginPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginPath, "installed_plugins"), nil
}
