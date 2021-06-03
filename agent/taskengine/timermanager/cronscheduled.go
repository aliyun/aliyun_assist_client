package timermanager

import (
	"time"

	cron "github.com/robfig/cron/v3"
)

// CronScheduled provides nextRun() interface for cron expression
type CronScheduled struct {
	schedule cron.Schedule
}

var (
	// Statically initialized cron expression parser, which should be goroutine-safe
	_cronExpressionParser cron.Parser
)

func init() {
	// Set supported fields of parser keeping compatibility with existing agent
	_cronExpressionParser = cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
}

// NewCronScheduled returns scheduler from cron expression
func NewCronScheduled(cronat string) (*CronScheduled, error) {
	schedule, err := _cronExpressionParser.Parse(cronat)
	if err != nil {
		return nil, err
	}
	return &CronScheduled{schedule: schedule}, nil
}

func (c *CronScheduled) NextRunFrom(t time.Time) (time.Duration, error) {
	nextRunTime := c.schedule.Next(t)
	return nextRunTime.Sub(t), nil
}

func (c *CronScheduled) nextRun() (time.Duration, error) {
	now := time.Now()
	return c.NextRunFrom(now)
}
