package timermanager

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/cronexpr"
)

// CronScheduled provides nextRun() interface for cron expression
type CronScheduled struct {
	expression *cronexpr.Expression
	location *time.Location

	isNoNextRun bool
}

var (
	CronYearFieldRegexp = regexp.MustCompile(`^\*$|^\?$|19[789][0-9]|20[0-9]{2}|^\*/(\d+)$`)
	// GMT+13:00/GMT+14:00 do exist, even GMT+13:45. But GMT-13:**/GMT-14:00 do not.
	GMTOffsetTimezoneRegexp = regexp.MustCompile(`GMT([+-])([0-9]|1[0-4]):([0-5][0-9])`)

	ErrInvalidCronExpression = newCronParameterError("InvalidCronExpression", "invalid cron expression cannot be parsed")
	ErrTimezoneInformationCorrupt = newCronParameterError("TimezoneInformationCorrupt", "Information of sepcified timezone in cron expression cannot be parsed")
	ErrInvalidGMTOffsetForTimezone = newCronParameterError("InvalidGMTOffsetForTimezone", "invalid GMT+-offset format at timezone field cannot be parsed")
	ErrInvalidGMTOffsetHourForTimezone = newCronParameterError("InvalidGMTOffsetHourForTimezone", "invalid hour value for GMT+-offset format at timezone field cannot be parsed")
	ErrInvalidGMTOffsetMinuteForTimezone = newCronParameterError("InvalidGMTOffsetMinuteForTimezone", "invalid minute value for GMT+-offset format at timezone field cannot be parsed")
	ErrCronExpressionExpired = newCronParameterError("CronExpressionExpired", "cron expression had expired at the moment of system clock")
)

// NewCronScheduled returns scheduler from cron expression
//
// Classic style and new styles of cron expression are both supported:
// * Classic: `Seconds Minutes Hours Day_of_month Month Day_of_week`
// * New: `Seconds Minutes Hours Day_of_month Month Day_of_week Year(optional) Timezone(optional)`
//
// For timezone specification, two formats are supported:
// * Complete name in TZ database, e.g., Asia/Shanghai, America/Los_Angeles
// * GMT offset (no leading zero in hour), e.g., GMT+8:00, GMT-6:00
// * Some fixed and unambiguous abbreviation names of timezone, namely:
//   + GMT
//   + UTC
func NewCronScheduled(cronat string) (*CronScheduled, error) {
	canonicalizedCronat, location, err := _splitExpressionAndLocation(cronat)
	if err != nil {
		return nil, err
	}

	expression, err := cronexpr.Parse(canonicalizedCronat)
	if err != nil {
		return nil, err
	}
	schedule := CronScheduled{
		expression: expression,
		location: location,
		isNoNextRun: false,
	}
	// Report cron expression expiration as early as possible
	if _, err := schedule.nextRun(); errors.Is(err, ErrNoNextRun) {
		return nil, ErrCronExpressionExpired
	}
	return &schedule, nil
}

func _splitExpressionAndLocation(cronat string) (string, *time.Location, error) {
	trimmedCronat := strings.TrimSpace(cronat)
	fields := strings.Fields(trimmedCronat)
	switch len(fields) {
	case 6:
		// Append wildcard year field if only 6 fields are present, i.e.,
		// classic style of cron expression, just for:
		// * making year field optional
		// * compatiblity with previous version of agent
		// * overwriting default modification behavior of gorhill/cronexpr library
		//   under such situation.
		return trimmedCronat + " *", nil, nil
	case 7:
		// If the last field satisfies any condition below, it would be
		// considered as the year field, and the whole expression is a valid
		// cron expression without the optional timezone field:
		// * it is asterisk wildcard character `\*` or question-mark wildcard
		//   character `\?`
		// * it contains year number, i.e., matches regexp
		//   `19[789][0-9]|20[0-9]{2}`
		// * it starts with asterisk wildcard character and interval delimiter
		//   slash, i.e., `\*/`
		// See https://github.com/gorhill/cronexpr/blob/master/cronexpr_parse.go
		// for concrete implementation of year field matching.
		if CronYearFieldRegexp.MatchString(fields[6]) {
			return trimmedCronat, nil, nil
		}
		// Otherwise the last field is considered as timezone specification, and
		// the original string is in classic style but with timezone field.
		location, err := _parseLocation(fields[6])
		if err != nil {
			return "", nil, err
		}
		fields[6] = "*"
		return strings.Join(fields, " "), location, nil
	case 8:
		location, err := _parseLocation(fields[7])
		if err != nil {
			return "", nil, err
		}
		return strings.Join(fields[:7], " "), location, nil
	default:
		return "", nil, ErrInvalidCronExpression
	}
}

// For the supported format of timezone specification, see documentation of
// NewCronScheduled function
func _parseLocation(tzSpec string) (*time.Location, error) {
	trimmedTZSpec := strings.TrimSpace(tzSpec)
	// For complete name in TZ database, e.g., Asia/Shanghai, America/Los_Angeles:
	if strings.Contains(trimmedTZSpec, "/") {
		location, err := time.LoadLocation(trimmedTZSpec)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrTimezoneInformationCorrupt, err.Error())
		}

		return location, nil
	}
	// * GMT offset (no leading zero in hour), e.g., GMT+8:00, GMT-6:00
	if strings.ContainsAny(trimmedTZSpec, "+-") {
		matches := GMTOffsetTimezoneRegexp.FindAllStringSubmatch(trimmedTZSpec, -1)
		if len(matches) != 1 {
			return nil, ErrInvalidGMTOffsetForTimezone
		}
		// Now the only match contains 4 items: [[<NAME>, <SIGN>, <HOUR>, <MINUTE>]]
		locationName := matches[0][0]
		offsetSign := matches[0][1]
		offsetHour, err := strconv.Atoi(matches[0][2])
		if err != nil {
			return nil, ErrInvalidGMTOffsetHourForTimezone
		}
		offsetMinute, err := strconv.Atoi(matches[0][3])
		if err != nil {
			return nil, ErrInvalidGMTOffsetMinuteForTimezone
		}
		offsetSeconds := offsetHour * 60 * 60 + offsetMinute * 60
		if offsetSign == "-" {
			offsetSeconds = -offsetSeconds
		}
		return time.FixedZone(locationName, offsetSeconds), nil
	}
	// For fixed and unambiguous abbreviation names of timezone
	if trimmedTZSpec == "GMT" || trimmedTZSpec == "UTC" {
		return time.UTC, nil
	}

	return nil, ErrTimezoneInformationCorrupt
}

func (c *CronScheduled) Location() *time.Location {
	return c.location
}

func (c *CronScheduled) NoNextRun() bool {
	return c.isNoNextRun
}

func (c *CronScheduled) NextRunFrom(t time.Time) (time.Duration, error) {
	nextRunTime := c.expression.Next(t)
	if nextRunTime.IsZero() {
		return time.Duration(-1), ErrNoNextRun
	}

	return nextRunTime.Sub(t), nil
}

func (c *CronScheduled) nextRun() (time.Duration, error) {
	now := time.Now()
	if c.location != nil {
		now = now.In(c.location)
	}

	timeToWait, err := c.NextRunFrom(now)
	if errors.Is(err, ErrNoNextRun) {
		c.isNoNextRun = true
	}
	return timeToWait, err
}
