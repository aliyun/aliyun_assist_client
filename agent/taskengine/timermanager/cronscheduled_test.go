package timermanager

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCronScheduled(t *testing.T) {
	const CronExpression = "*/20 * * * * ?"

	scheduled, err := NewCronScheduled(CronExpression)
	assert.NoError(t, err, "NewCronScheduled should correctly parse specified cron expression")

	expectedSchedule, err := _cronExpressionParser.Parse(CronExpression)
	assert.NoError(t, err, "_cronExpressionParser.Parse should not raise error")

	testTime := time.Now()
	assert.Exactly(t, expectedSchedule.Next(testTime),
		scheduled.schedule.Next(testTime),
		"CronScheduled should generate same time of next schedule for same cron expression")
}

func TestNextRunFrom(t *testing.T) {
	const CronExpression = "*/20 * * * * ?"
	expectedSchedule, err := _cronExpressionParser.Parse(CronExpression)
	assert.NoError(t, err, "_cronExpressionParser.Parse should not raise error")

	scheduled, _ := NewCronScheduled(CronExpression)
	testTime := time.Now()
	expectedDuration := expectedSchedule.Next(testTime).Sub(testTime)
	next, err := scheduled.nextRunFrom(testTime)
	assert.NoError(t, err, "nextRunFrom should not return error")
	assert.Exactly(t, expectedDuration, next, "nextRunFrom should returns same time of next schedule from same timestamp for same cron expression")
}
