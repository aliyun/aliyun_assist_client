package timetool

import (
	"time"
)

// GetAccurateTime returns current time in millisecond precision
func GetAccurateTime() int64 {
	now := time.Now().Local()
	return ToAccurateTime(now)
}

// ToAccurateTime returns specified time object in millisecond precision
func ToAccurateTime(t time.Time) int64 {
	return t.Unix() * 1000 + int64(t.Nanosecond() / 1000000)
}

func ToStableElapsedTime(t time.Time, base time.Time) time.Time {
	// Time-measuring opeartion between time.Time objects, such as subtractions,
	// produces monotonic clock reading of elapsed time
	elapsedDuration := t.Sub(base)
	// (time.Time).Add adds the same duration to both the wall clock and
	// monotonic clock readings to compute the result, which produces end time
	// consistent with start time in time settings.
	monotonicWallTime := base.Add(elapsedDuration)
	// Should get t - base == elapsedDuration in wall clock reading
	// finally.
	return monotonicWallTime
}

func ApiTimeFormat(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

func ParseApiTime(s string) (t time.Time, err error) {
	t, err = time.ParseInLocation("2006-01-02T15:04:05Z", s, time.UTC)
	return
}

func UtcNowStr() string {
	return ApiTimeFormat(time.Now())
}
