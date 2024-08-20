//+build windows

package pathutil

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	gomonkey "github.com/agiledragon/gomonkey/v2"
)

func TestGetWinXxxPath(t *testing.T) {
	guard_1 := gomonkey.ApplyFunc(os.Executable, func() (string, error) {
		return "C:\\ProgramData\\aliyun\\assist\\2.1.3.999\\aliyun_assist_service.exe", nil
	})
	defer guard_1.Reset()
	scriptPath, _ := GetScriptPath()
	hybridPath, _ := GetHybridPath()
	configPath, _ := GetConfigPath()
	crossVersionConfigPath, _ := GetCrossVersionConfigPath()
	cachePath, _ := GetCachePath()
	pluginPath, _ := GetPluginPath()
	assert.Equal(t, "C:\\ProgramData\\aliyun\\assist\\work\\script", scriptPath)
	assert.Equal(t, "C:\\ProgramData\\aliyun\\assist\\hybrid", hybridPath)
	assert.Equal(t, "C:\\ProgramData\\aliyun\\assist\\2.1.3.999\\config", configPath)
	assert.Equal(t, "C:\\ProgramData\\aliyun\\assist\\config", crossVersionConfigPath)
	assert.Equal(t, "C:\\ProgramData\\aliyun\\assist\\cache", cachePath)
	assert.Equal(t, "C:\\ProgramData\\aliyun\\assist\\plugin", pluginPath)
}
