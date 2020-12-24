package timetool

import (
	"time"
)

// GetAccurateTime returns current time in millisecond precision
func GetAccurateTime() int64 {
	now := time.Now().Local()
	return now.Unix() * 1000 + int64(now.Nanosecond() / 1000000)
}
