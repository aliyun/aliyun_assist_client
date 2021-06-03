package statemanager

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

func removeCacheDir(p string) {
	os.RemoveAll(p)
}

func TestConfigCache(t *testing.T) {
	p, _ := util.GetCachePath()
	removeCacheDir(p)
	defer removeCacheDir(p)

	r, err := LoadConfigCache()
	assert.Nil(t, err)
	assert.Nil(t, r)

	r = &ListInstanceStateConfigurationsResult{
		RequestId:  "AAAA",
		Changed:    true,
		Checkpoint: "2020-12-18T00:00:00Z",
		StateConfigurations: []StateConfiguration{{
			StateConfigurationId: "sc-abc",
			TemplateName:         "demo-template",
			TemplateVersion:      "v1",
			Parameters:           "{\"key\":\"value\"}",
			ConfigureMode:        "ApplyAndAutoCorrect",
			ScheduleType:         "cron",
			ScheduleExpression:   ""}}}
	err = WriteConfigCache(r)
	assert.Nil(t, err)

	cached, err := LoadConfigCache()
	assert.Nil(t, err)
	assert.NotNil(t, cached)
	assert.Equal(t, *r, *cached)
}

func TestTemplateCache(t *testing.T) {
	p, _ := util.GetCachePath()
	removeCacheDir(p)
	defer removeCacheDir(p)
	
	data, err := LoadTemplateCache("hostsConfig", "v1")
	assert.Nil(t, data)
	assert.Nil(t, err)

	content := `{
		"FormatVersion": "OOS-2019-06-01-State",
		"Description": "示例模板",
		"Parameters": {
		  "mode": {
			"Type": "String",
			"Default": "644"
		  }
		},
		"States": [
		  {
			"ResourceType": "ACS:File",
			"Properties": {
			  "Ensure": "Present",
			  "State": "File",
			  "Mode": "{{ mode }}",
			  "DestinationPath": "/etc/hosts",
			  "SourcePath": "http://the-source-of-file"
			}
		  }
		]
	}`

	err = WriteTemplateCache("hostsConfig", "v1", []byte(content))
	assert.Nil(t, err)

	data, err = LoadTemplateCache("hostsConfig", "v1")
	assert.Equal(t, []byte(content), data)
	assert.Nil(t, err)
}
