package statemanager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMode1(t *testing.T) {
	config := StateConfiguration{
	StateConfigurationId : "sc-001",
	TemplateName         : "linux-host-file",
	TemplateVersion      : "1.0",
	Parameters           : `{"owner": "root"}`,
	ConfigureMode        : "ApplyAndMonitor",
	ScheduleType         : "cron",
	ScheduleExpression   : "0 0 */1 ? * *",
	// SuccessfulApplyTime  : "2020-10-20T00:00:00Z",
	DefinitionUpdateTime : "2020-10-21T00:00:00Z",
	}
	mode := getMode(config)
	assert.Equal(t, Apply, mode)
}

func TestGetMode2(t *testing.T) {
	config := StateConfiguration{
	StateConfigurationId : "sc-001",
	TemplateName         : "linux-host-file",
	TemplateVersion      : "1.0",
	Parameters           : `{"owner": "root"}`,
	ConfigureMode        : "ApplyAndMonitor",
	ScheduleType         : "cron",
	ScheduleExpression   : "0 0 */1 ? * *",
	SuccessfulApplyTime  : "2020-10-22T00:00:00Z",
	DefinitionUpdateTime : "2020-10-21T00:00:00Z",
	}
	mode := getMode(config)
	assert.Equal(t, Monitor, mode)
}

func TestGetMode3(t *testing.T) {
	config := StateConfiguration{
	StateConfigurationId : "sc-001",
	TemplateName         : "linux-host-file",
	TemplateVersion      : "1.0",
	Parameters           : `{"owner": "root"}`,
	ConfigureMode        : "ApplyAndMonitor",
	ScheduleType         : "cron",
	ScheduleExpression   : "0 0 */1 ? * *",
	SuccessfulApplyTime  : "2020-10-22T00:00:00Z",
	DefinitionUpdateTime : "2020-10-23T00:00:00Z",
	}
	mode := getMode(config)
	assert.Equal(t, Apply, mode)
}
