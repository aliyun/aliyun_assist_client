// +build darwin freebsd linux netbsd openbsd

package application

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

var (
	sampleData = `{"Name":"` + mark(`aliyun-OOS-agent`) + `","Version":"` + mark(`1.2.0.0-1`) +
		`","Release":"` + mark(`1`) + `","Epoch":"` + mark(`(none)`) +
		`","Publisher":"` + mark(`aliyun.com, Inc. "<ecs-OOS-feedback@aliyun.com>"`) +
		`","ApplicationType":"` + mark(`admin`) + `","Architecture":"` + mark(`amd64`) + `","Url":"","Summary":"` +
		mark(`Description with "quotes" 'and' `+"tabs\t"+` and
		new lines`) + `","PackageId":"` + mark(`aliyun-OOS-agent_1.2_amd64.rpm`) + `"},` +

		`{"Name":"` + mark(`adduser`) + `","Version":"` + mark(`3.113+nmu3ubuntu3`) + `","Publisher":"` +
		mark(`Ubuntu Core Developers <ubuntu-devel-discuss@lists.ubuntu.com>`) + `","Release":"` + mark(`9.amzn2`) + `","Epoch":"` + mark(`14`) +
		`","ApplicationType":"` + mark(`admin`) + `","Architecture":"` + mark(`all`) +
		`","Url":"` + mark(`http://alioth.debian.org/projects/adduser/`) + `",` +
		`"Summary":"` + mark(`add and remove users and groups
 This package includes the 'adduser' and 'deluser' commands for creating
 and removing users.`) + `","PackageId":"` + mark(`adduser_3.113+nmu3ubuntu4_all.deb`) + `"},` +

		`{"Name":"` + mark(`"sed"`) + `","Publisher":"` + mark(`"aliyun.com"`) + `","Version":"` + mark(`"4.2.1"`) +
		`","InstalledTime":"` + mark(`1454346676`) + `",` +
		`"ApplicationType":"` + mark(`"Applications/Text"`) + `","Architecture":"` + mark(`"x86_64"`) + `","Url":"` +
		mark(`"http://sed.sourceforge.net/"`) + `",` + `"Summary":"` + mark(`A GNU "stream" text editor`) + `","PackageId":"` +
		mark(`"sed-4.2.1-7.9.amzn1.src.rpm"`) + `"},` +

		`{"Name":"` + mark(`sed`) + `","Version":"` + mark(`4.2.2-7`) + `","Publisher":"` + mark(`Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>`) +
		`","ApplicationType":"` + mark(`utils`) + `","Architecture":"` + mark(`amd64`) + `","Url":"` + mark(`http://www.gnu.org/software/sed/`) + `",` +
		`"Summary":"` + mark(`The GNU sed stream editor
sed reads the specified files or the standard input if no
files are specified, makes editing changes according to a
list of commands, and writes the results to the standard
output.`) + `","PackageId":"` + mark(`sed_4.2.2-7_amd64.deb`) + `"},` +

		`{"Name":"` + mark(`vim-filesystem`) + `","Version":"` + mark(`8.0.0503`) + `","Publisher":"` +
		mark(`aliyun.com`) + `","Release":"` + mark(`1.45.amzn1`) + `","Epoch":"` + mark(`(none)`) +
		`","ApplicationType":"` + mark(`Applications/Editors`) + `","Architecture":"` + mark(`x86_64`) +
		`","Url":"` + mark(`http://www.vim.org/`) + `",` +
		`"Summary":"` + mark(`VIM filesystem layout`) + `","PackageId":"` + mark(`vim-6:8.0.0503-1.45.amzn1.src.rpm`) + `"},`
)

var snapSampleData = "Name  Version    Rev   Tracking  Publisher   Notes\ncore  16-2.43.3  8689  stable    canonical*  core\n"

var sampleDataParsed = []model.ApplicationData{
	{
		Name:            "aliyun-OOS-agent",
		Version:         "1.2.0.0-1",
		Release:         "1",
		Epoch:           "",
		Publisher:       "aliyun.com, Inc. \"<ecs-OOS-feedback@aliyun.com>\"",
		ApplicationType: "admin",
		Architecture:    "x86_64",
		URL:             "",
		Summary:         "Description with \"quotes\" 'and' tabs\t and",
		PackageId:       "aliyun-OOS-agent_1.2_amd64.rpm",
	},
	{
		Name:            "adduser",
		Version:         "3.113+nmu3ubuntu3",
		Release:         "9.amzn2",
		Epoch:           "14",
		Publisher:       "Ubuntu Core Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
		ApplicationType: "admin",
		Architecture:    "all",
		URL:             "http://alioth.debian.org/projects/adduser/",
		Summary:         "add and remove users and groups",
		PackageId:       "adduser_3.113+nmu3ubuntu4_all.deb",
	},
	{
		Name:            "\"sed\"",
		Version:         "\"4.2.1\"",
		Release:         "",
		Epoch:           "",
		Publisher:       "\"aliyun.com\"",
		InstalledTime:   "2016-02-01T17:11:16Z",
		ApplicationType: "\"Applications/Text\"",
		Architecture:    "\"x86_64\"",
		URL:             "\"http://sed.sourceforge.net/\"",
		Summary:         "A GNU \"stream\" text editor",
		PackageId:       "\"sed-4.2.1-7.9.amzn1.src.rpm\"",
	},
	{
		Name:            "sed",
		Version:         "4.2.2-7",
		Release:         "",
		Epoch:           "",
		Publisher:       "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
		ApplicationType: "utils",
		Architecture:    "x86_64",
		URL:             "http://www.gnu.org/software/sed/",
		Summary:         "The GNU sed stream editor",
		PackageId:       "sed_4.2.2-7_amd64.deb",
	},
	{
		Name:            "vim-filesystem",
		Version:         "8.0.0503",
		Release:         "1.45.amzn1",
		Epoch:           "6",
		Publisher:       "aliyun.com",
		ApplicationType: "Applications/Editors",
		Architecture:    "x86_64",
		URL:             "http://www.vim.org/",
		Summary:         "VIM filesystem layout",
		PackageId:       "vim-6:8.0.0503-1.45.amzn1.src.rpm",
	},
}

var snapSampleDataParsed = `{"Name":"` + mark(`core`) + `","Publisher":"` + mark(`canonical*`) + `","Version":"` + mark(`16-2.43.3`) +
	`","ApplicationType":"` + mark(`admin`) + `","Architecture":"` + mark(``) + `","Url":"` + mark(``) + `",` +
	`"Summary":"` + mark(``) + `","PackageId":"` + mark(``) + `"}`

func MockTestExecutorWithError(command string, args ...string) ([]byte, error) {
	var result []byte
	return result, fmt.Errorf("random error")
}

func MockTestExecutorWithoutError(command string, args ...string) ([]byte, error) {
	return []byte(sampleData), nil
}

func TestGetApplicationData(t *testing.T) {

	var data []model.ApplicationData
	var err error
	mockCommand := "RandomCommand"
	mockArgs := []string{
		"RandomArgument-1",
		"RandomArgument-2",
	}

	//testing with error
	cmdExecutor = MockTestExecutorWithError

	data, err = getApplicationData(mockCommand, mockArgs)

	assert.NotNil(t, err, "Error must be thrown when command execution fails")
	assert.Equal(t, 0, len(data), "When command execution fails - application dataset must be empty")

	//testing without error
	cmdExecutor = MockTestExecutorWithoutError

	data, err = getApplicationData(mockCommand, mockArgs)

	assert.Nil(t, err, "Error must not be thrown with MockTestExecutorWithoutError")
	assert.Equal(t, sampleDataParsed, data)
}
