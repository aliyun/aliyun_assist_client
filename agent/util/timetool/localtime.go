package timetool

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	UnknownTimezoneName = "unknown"
)

var (
	// The time stdlib in golang ONLY initializes timezone information for local
	// system clock when program starts, after which changes of timezone setting
	// in system would not be available in program, until it restarts.
	// See below issues on GitHub for related information:
	// * https://github.com/golang/go/issues/46417
	// * https://github.com/golang/go/issues/28020
	systemTimezoneName           string
	detectSystemTimezoneNameOnce sync.Once

	ErrDetectSystemTimezoneName = errors.New("failed to detect TZ name in system")
)

func NowWithTimezoneName() (time.Time, int, string) {
	now := time.Now()
	// FIXME: Detected system timezone setting after a while may not be the same
	// as that one detected by golang stdlib at initalization.
	detectSystemTimezoneNameOnce.Do(func() {
		timezoneName, err := GetCurrentTimezoneName()
		// Fallback to GMT+-offset/UTC if error encountered
		if err != nil {
			timezoneAbbr, offset := now.Zone()
			log.GetLogger().WithFields(logrus.Fields{
				"currentTime":          now.Format(time.RFC3339),
				"currentTimezoneAbbr":  timezoneAbbr,
				"currentOffsetFromUTC": offset,
			}).WithError(err).Warning("Failed to detect canonical name of current system TZ setting, fallback to GMT+-offset value from golang stdlib")
			return
		}

		systemTimezoneName = timezoneName
	})

	_, currentOffsetFromUTC := now.Zone()
	timezoneName := systemTimezoneName
	if timezoneName == "" {
		offset := currentOffsetFromUTC
		if offset == 0 {
			timezoneName = "UTC"
		} else {
			offsetHourWithSign := offset / 3600
			if offset < 0 {
				offset = -offset
			}
			offsetMinute := (offset % 3600) / 60
			timezoneName = fmt.Sprintf("GMT%+d:%02d", offsetHourWithSign, offsetMinute)
		}
	}
	return now, currentOffsetFromUTC, timezoneName
}
