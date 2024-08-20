package acspluginmanager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/stretchr/testify/assert"
)

func TestInstalledPluginsDb(t *testing.T) {
	guard_plugindir := gomonkey.ApplyFunc(pathutil.GetPluginPath, func() (string, error) {
		curr, err := os.Executable()
		assert.Nil(t, err)
		curr = filepath.Dir(curr)
		pluginDir := filepath.Join(curr, "plugin_test_dir")
		return pluginDir, nil
	})
	defer guard_plugindir.Reset()

	pluginDir, _ := pathutil.GetPluginPath()
	os.MkdirAll(pluginDir, os.ModePerm)
	defer os.RemoveAll(pluginDir)

	
}