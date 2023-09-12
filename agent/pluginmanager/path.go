package pluginmanager

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

func getInstalledPluginsBoltPath() (string, error) {
	pluginPath, err := pathutil.GetPluginPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginPath, "installed_plugins.db"), nil
}

func getInstalledPluginsJSONPath() (string, error) {
	pluginPath, err := pathutil.GetPluginPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginPath, "installed_plugins"), nil
}
