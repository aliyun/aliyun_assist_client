//+build !windows 

package pathutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	gomonkey "github.com/agiledragon/gomonkey/v2"
)


func TestGetCurrentPath(t *testing.T) {
	path,err := GetCurrentPath()
	assert.DirExists(t, path)
	assert.Equal(t, true, err==nil)
}

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
	hybridPath, _ := GetHybridPath()
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
