package timermanager

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type RateScheduled struct {
	startTime time.Time
	period time.Duration
}

const (
	RateExpressionPrefix = "rate("

	frequencyUpperLimit = time.Duration(1000 * 24) * time.Hour
)

var (
	rateExpressionRegexp = regexp.MustCompile(`(?i)(rate\s*\((\d+)\s*([smhd])\))`)

	ErrInvalidRateExpression = newCronParameterError("InvalidRateExpression", "invalid rate expression cannot be parsed")
	ErrRateFrequencyTooLarge = newCronParameterError("RateFrequencyTooLarge", "specified frequency in rate expression is too large, i.e., larger than 1,000 days")
)

// NewRateScheduled returns scheduler from rate() expression
func NewRateScheduled(cronat string, startTime time.Time) (*RateScheduled, error) {
	matches := rateExpressionRegexp.FindAllStringSubmatch(cronat, -1)
	if len(matches) != 1 {
		return nil, ErrInvalidRateExpression
	}

	match := matches[0]
	if len(match) != 4 {
		return nil, ErrInvalidRateExpression
	}
	if match[1] != cronat {
		return nil, ErrInvalidRateExpression
	}

	value, err := strconv.ParseInt(match[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidRateExpression, err.Error())
	}
	if value == 0 {
		return nil, fmt.Errorf("%w: Period value should be positive number", ErrInvalidRateExpression)
	}

	unit := strings.ToLower(match[3])
	var period time.Duration
	if unit == "s" {
		period = time.Duration(value) * time.Second
	} else if unit == "m" {
		period = time.Duration(value) * time.Minute
	} else if unit == "h" {
		period = time.Duration(value) * time.Hour
	} else if unit == "d" {
		period = time.Duration(value * 24) * time.Hour
	} else {
		return nil, fmt.Errorf("%w: Invalid period unit", ErrInvalidRateExpression)
	}

	if period > frequencyUpperLimit {
		return nil, ErrRateFrequencyTooLarge
	}

	return &RateScheduled{
		startTime: startTime,
		period: period,
	}, nil
}

func (r *RateScheduled) NextRunFrom(t time.Time) (time.Duration, error) {
	nextRunTime, err := r.scheduleNextRunTimeFrom(t)
	if err != nil {
		return time.Duration(-1), err
	}

	return nextRunTime.Sub(t), nil
}

func (r *RateScheduled) nextRun() (time.Duration, error) {
	now := time.Now()
	return r.NextRunFrom(now)
}

func (r *RateScheduled) scheduleNextRunTimeFrom(t time.Time) (time.Time, error) {
	// Special case: if the time base, t, is before the start time of rate
	// scheduler, just return the time for the 1st invocation, i.e., the end of
	// the 1st period after the creation time of task
	if t.Before(r.startTime) {
		return r.startTime.Add(r.period), nil
	}

	// Calculate the next time point from the start
	passedDuration := t.Sub(r.startTime)
	passedPeriods := int64(passedDuration / r.period)
	nextRunTime := r.startTime.Add(time.Duration(passedPeriods + 1) * r.period)
	return nextRunTime, nil
}
