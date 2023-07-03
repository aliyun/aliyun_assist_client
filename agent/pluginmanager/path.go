package pluginmanager

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func getInstalledPluginsJSONPath() (string, error) {
	pluginPath, err := util.GetPluginPath()
	if err != nil {
		return "", err
	}

	return filepath.Join(pluginPath, "installed_plugins"), nil
}
