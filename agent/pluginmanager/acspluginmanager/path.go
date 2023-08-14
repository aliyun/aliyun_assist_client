package acspluginmanager

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func getPluginManagerCachePath() (string, error) {
	cachePath, err := util.GetCachePath()
	if err != nil {
		return "", err
	}

	pluginManagerCachePath := filepath.Join(cachePath, "plugin-manager")
	err = util.MakeSurePath(pluginManagerCachePath)
	return pluginManagerCachePath, err
}
