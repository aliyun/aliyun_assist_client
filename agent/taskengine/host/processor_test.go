package host

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/stretchr/testify/assert"
)

// TestCleanup 测试 HostProcessor 的 Cleanup 方法
func TestCleanup(t *testing.T) {
	mockScriptFile := filepath.Join(os.TempDir(), "mock_script.sh")
	mockProcess := &HostProcessor{
		scriptFilePath: mockScriptFile,
	}
	mockIsPeriodic := false
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(mockProcess), "isPeriodic", func(*HostProcessor) bool {
		return mockIsPeriodic
	}).Reset()
	defer os.Remove(mockScriptFile)

	testCase := []struct {
		Name             string
		IsPeriodic       bool
		RemoveScriptFile bool
	}{
		{
			Name:             "Once_RemoveScriptFile_True",
			IsPeriodic:       false,
			RemoveScriptFile: true,
		},
		{
			Name:             "Once_RemoveScriptFile_False",
			IsPeriodic:       false,
			RemoveScriptFile: false,
		},
		{
			Name:             "Periodic_RemoveScriptFile_True",
			IsPeriodic:       true,
			RemoveScriptFile: true,
		},
		{
			Name:             "Periodic_RemoveScriptFile_False",
			IsPeriodic:       true,
			RemoveScriptFile: false,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			os.WriteFile(mockScriptFile, []byte("mock script content"), 0644)
			mockIsPeriodic = tc.IsPeriodic
			err := mockProcess.Cleanup(tc.RemoveScriptFile)
			assert.Nil(t, err)
			assert.Nil(t, mockProcess.processCmd)

			scriptExist := fileutil.CheckFileIsExist(mockScriptFile)
			switch tc.Name {
			case "Once_RemoveScriptFile_True":
				assert.False(t, scriptExist)
			case "Once_RemoveScriptFile_False":
				assert.True(t, scriptExist)
			case "Periodic_RemoveScriptFile_True":
				assert.True(t, scriptExist)
			case "Periodic_RemoveScriptFile_False":
				assert.True(t, scriptExist)
			}
		})
	}
}

// TestCancel 测试 HostProcessor 的 Cancel 方法
func TestCancel(t *testing.T) {
	mockScriptFile := filepath.Join(os.TempDir(), "mock_script.sh")
	mockIsPeriodic := false
	mockProcess := &HostProcessor{}
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(mockProcess), "isPeriodic", func(*HostProcessor) bool {
		return mockIsPeriodic
	}).Reset()
	defer os.Remove(mockScriptFile)

	testCase := []struct {
		Name           string
		Canceled       bool
		IsPeriodic     bool
		KeepScriptFile bool
	}{
		{
			Name:           "Uncanceled_Once_KeepScriptFile_True",
			Canceled:       false,
			IsPeriodic:     false,
			KeepScriptFile: true,
		},
		{
			Name:           "Uncanceled_Once_KeepScriptFile_False",
			Canceled:       false,
			IsPeriodic:     false,
			KeepScriptFile: false,
		},
		{
			Name:           "Uncanceled_Periodic_KeepScriptFile_True",
			Canceled:       false,
			IsPeriodic:     true,
			KeepScriptFile: true,
		},
		{
			Name:           "Uncanceled_Periodic_KeepScriptFile_False",
			Canceled:       false,
			IsPeriodic:     true,
			KeepScriptFile: false,
		},
		{
			Name:           "Canceled_Once_KeepScriptFile_True",
			Canceled:       false,
			IsPeriodic:     false,
			KeepScriptFile: true,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.Name, func(t *testing.T) {
			defer gomonkey.ApplyFunc(flagging.GetTaskKeepScriptFile, func() bool {
				return tc.KeepScriptFile
			}).Reset()

			os.WriteFile(mockScriptFile, []byte("mock script content"), 0644)
			mockIsPeriodic = tc.IsPeriodic
			mockProcess := &HostProcessor{
				scriptFilePath: mockScriptFile,
				processCmd:     &process.ProcessCmd{},
			}

			err := mockProcess.Cancel()
			assert.Nil(t, err)
			assert.Nil(t, mockProcess.processCmd)
			assert.True(t, mockProcess.canceled)
			scriptExist := fileutil.CheckFileIsExist(mockScriptFile)

			switch tc.Name {
			case "Uncanceled_Once_KeepScriptFile_True":
				assert.True(t, scriptExist)
			case "Uncanceled_Once_KeepScriptFile_False":
				// Once task, script file will be delete in HostProcessor.Cleanup()
				assert.True(t, scriptExist)
			case "Uncanceled_Periodic_KeepScriptFile_True":
				assert.True(t, scriptExist)
			case "Uncanceled_Periodic_KeepScriptFile_False":
				assert.False(t, scriptExist)
			case "Canceled_Once_KeepScriptFile_False":
				// HostProcessor.Cancel() should return directly
				assert.True(t, scriptExist)
			}
		})
	}
}
