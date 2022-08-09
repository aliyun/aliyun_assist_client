package timermanager

import (
	"strings"
	"time"
)

type AtScheduled struct {
	schedule time.Time
}

const (
	AtExpressionPrefix = "at("

	rfc3339WithoutTimezone = "2006-01-02T15:04:05"
)

var (
	ErrInvalidAtExpression = newCronParameterError("InvalidAtExpression", "invalid at expression cannot be parsed")
	ErrAtExpressionExpired = newCronParameterError("AtExpressionExpired", "at expression had expired at the moment of system clock")
)

// NewAtScheduled returns scheduler from at expression
func NewAtScheduled(cronat string) (*AtScheduled, error) {
	exprLength := len(cronat)
	if exprLength < 4 {
		return nil, ErrInvalidAtExpression
	}
	if strings.ToLower(cronat[:len(AtExpressionPrefix)]) == AtExpressionPrefix && cronat[exprLength - 1] == ')' {
		cronat = cronat[3:exprLength - 1]
	}

	schedule, err := time.Parse(rfc3339WithoutTimezone, cronat)
	if err != nil {
		return nil, err
	}
	return &AtScheduled{schedule: schedule}, nil
}

func (a *AtScheduled) NextRunFrom(t time.Time) (time.Duration, error) {
	utcTime := t.UTC()
	if utcTime.After(a.schedule) {
		return time.Duration(-1), ErrAtExpressionExpired
	}

	return a.schedule.Sub(utcTime), nil
}

func (a *AtScheduled) nextRun() (time.Duration, error) {
	now := time.Now()
	return a.NextRunFrom(now)
}
