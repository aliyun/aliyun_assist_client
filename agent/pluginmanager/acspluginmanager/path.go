package acspluginmanager

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

func getPluginManagerCachePath() (string, error) {
	cachePath, err := pathutil.GetCachePath()
	if err != nil {
		return "", err
	}

	pluginManagerCachePath := filepath.Join(cachePath, "plugin-manager")
	err = pathutil.MakeSurePath(pluginManagerCachePath)
	return pluginManagerCachePath, err
}
