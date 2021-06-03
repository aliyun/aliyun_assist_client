package windowsupdate

import (
	"encoding/json"
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testUpdate = []model.WindowsUpdateData{
	{
		HotFixId:      "KB000001",
		Description:   "Security Update",
		InstalledTime: "Wednesday, October 15, 2014 12:00:00 AM",
		InstalledBy:   "ADMINISTRATOR",
	},
	{
		HotFixId:      "KB000002",
		Description:   "Update",
		InstalledTime: "Friday, June 20, 2014 12:00:00 AM",
		InstalledBy:   "NT AUTHORITY SYSTEM",
	},
}

func testExecuteCommand(command string, args ...string) ([]byte, error) {

	output, _ := json.Marshal(testUpdate)
	return output, nil
}

func testExecuteCommandEmpty(command string, args ...string) ([]byte, error) {

	return make([]byte, 0), nil
}
func TestExecuteCommand(t *testing.T) {
	out, _ := executeCommand("shell", "echo hello")
	// assert.Nil(t, err.Error())
	assert.Equal(t, "", string(out))
}

func TestGather(t *testing.T) {
	c := model.Config{}
	g := Garherer()
	cmdExecutor = testExecuteCommand
	item, err := g.Run(c)
	if err != nil {
		print(err.Error())
	}
	assert.Equal(t, 1, len(item))
	assert.Equal(t, GathererName, item[0].Name)
	assert.Equal(t, SchemaVersionOfWindowsUpdate, item[0].SchemaVersion)
	assert.Equal(t, SchemaVersionOfWindowsUpdate, item[0].SchemaVersion)
	assert.Equal(t, testUpdate, item[0].Content)
}
