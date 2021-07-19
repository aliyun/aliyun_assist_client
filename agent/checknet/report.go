package checknet

import (
	"time"
)

const (
	_refreshNetcheckTimeThreshold = time.Duration(15) * time.Minute
)

type CheckReport struct {
	Result int
	FinishedTime time.Time
}

func isReportOutdated(reportedTime time.Time) bool {
	return time.Now().Local().Sub(reportedTime.Local()) >= _refreshNetcheckTimeThreshold
}
