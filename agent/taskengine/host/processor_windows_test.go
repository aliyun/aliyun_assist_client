package host

import (
	"errors"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/langutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

// Hangul, an emoji and an umlaut are all missing from GBK
const nonGBKScript = "'a한b😀cäd中'"

func newProcessorOnChineseWindows(t *testing.T, commandType string) *HostProcessor {
	patches := gomonkey.ApplyFuncReturn(langutil.GetDefaultLang, uint32(0x804))
	patches.ApplyFuncReturn(pathutil.GetScriptPath, t.TempDir(), nil)
	// Prepare runs Set-ExecutionPolicy on this machine through SyncRunSimple
	patches.ApplyMethodReturn(&process.ProcessCmd{}, "SyncRunSimple", nil)
	t.Cleanup(patches.Reset)
	return &HostProcessor{TaskId: "t-test", CommandType: commandType}
}

func TestPrepareSavesPowerShellScriptAsUTF8WithBOM(t *testing.T) {
	p := newProcessorOnChineseWindows(t, "RunPowerShellScript")

	assert.Nil(t, p.Prepare(nonGBKScript))

	saved, err := os.ReadFile(p.scriptFilePath)
	assert.Nil(t, err)
	assert.Equal(t, append([]byte{0xEF, 0xBB, 0xBF}, nonGBKScript...), saved)
}

func TestPrepareSavesScriptForLauncherInGBK(t *testing.T) {
	p := newProcessorOnChineseWindows(t, "RunPowerShellScript")
	p.Launcher = "cscript.exe //Nologo {{ACS::ScriptFileName|Ext(.vbs)}}"

	assert.Nil(t, p.Prepare(`WScript.Echo "中"`))

	saved, err := os.ReadFile(p.scriptFilePath)
	assert.Nil(t, err)
	assert.Equal(t, []byte("WScript.Echo \"\xd6\xd0\""), saved)
}

func TestPrepareRejectsBatchScriptWithCharactersGBKLacks(t *testing.T) {
	p := newProcessorOnChineseWindows(t, "RunBatScript")

	err := p.Prepare(nonGBKScript)

	var execErr taskerrors.ExecutionError
	assert.True(t, errors.As(err, &execErr))
	assert.Equal(t, taskerrors.NewSaveScriptFileError(nil).ErrCode(), execErr.ErrCode())
	assert.Empty(t, p.scriptFilePath)
}
