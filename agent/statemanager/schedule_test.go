package statemanager

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/stretchr/testify/assert"
)

func TestRateInvalid(t *testing.T) {
	_, err := GetRateInSeconds("5minute")
	assert.NotNil(t, err)
	_, err = GetRateInSeconds("12:10")
	assert.NotNil(t, err)
}

func TestRefreshStateConfigTimers(t *testing.T) {
	var config1 = StateConfiguration{
		StateConfigurationId: "sc-config1",
		ScheduleType        : "rate",
		ScheduleExpression   : "30 minutes",
	}
	var config2 = StateConfiguration{
		StateConfigurationId: "sc-config2",
		ScheduleType        : "cron",
		ScheduleExpression   : "0 15 10 ? * *",
	}
	changed := isScheduleChanged(config1)
	assert.True(t, changed)
	changed = isScheduleChanged(config2)
	assert.True(t, changed)

	timermanager.InitTimerManager()
	setupStateConfigTimer(config1)
	setupStateConfigTimer(config2)
	assert.Equal(t, len(stateConfigTimers), 2)
	assert.Contains(t, stateConfigTimers, config1.StateConfigurationId)
	assert.Contains(t, stateConfigTimers, config2.StateConfigurationId)
	stateConfigTimer2 := stateConfigTimers[config2.StateConfigurationId]
	assert.Equal(t, stateConfigTimer2.scheduleType, config2.ScheduleType)
	assert.Equal(t, stateConfigTimer2.scheduleExpression, config2.ScheduleExpression)

	config2.ScheduleExpression = "0 0 12 * * ?"
	var config3 = StateConfiguration{
		StateConfigurationId: "sc-config3",
		ScheduleType        : "cron",
		ScheduleExpression   : "0 0/40 9-17 * * ?",
	}
	changed = isScheduleChanged(config1)
	assert.False(t, changed)
	changed = isScheduleChanged(config2)
	assert.True(t, changed)
	changed = isScheduleChanged(config3)
	assert.True(t, changed)

	refreshStateConfigTimers([]StateConfiguration{config2, config3})
	assert.Equal(t, len(stateConfigTimers), 2)
	assert.NotContains(t, stateConfigTimers, config1.StateConfigurationId)
	assert.Contains(t, stateConfigTimers, config2.StateConfigurationId)
	assert.Contains(t, stateConfigTimers, config3.StateConfigurationId)
	stateConfigTimer2 = stateConfigTimers[config2.StateConfigurationId]
	assert.Equal(t, stateConfigTimer2.scheduleType, config2.ScheduleType)
	assert.Equal(t, stateConfigTimer2.scheduleExpression, config2.ScheduleExpression)
	stateConfigTimer3 := stateConfigTimers[config3.StateConfigurationId]
	assert.Equal(t, stateConfigTimer3.scheduleType, config3.ScheduleType)
	assert.Equal(t, stateConfigTimer3.scheduleExpression, config3.ScheduleExpression)
}