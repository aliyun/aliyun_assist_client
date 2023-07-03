package checknet

import (
	"sync/atomic"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	heavylock "github.com/viney-shih/go-lock"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/atomicutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/networkcategory"
	"github.com/aliyun/aliyun_assist_client/agent/util/wrapgo"
)

var (
	// Atomic value for boolean indicator of whether it is need to report network
	// diagnostic result
	_needToReport atomicutil.AtomicBoolean
	// Atomic counter of how many times the network diagnostic result must be
	// reported due to force-to-report request
	_forceToReportResponses atomicutil.AtomicInt32

	// Atomic value for pointer of last network diagnostic report
	_neverDirectRW_atomic_lastReportPtr atomic.Value
	// _refreshingReportLock indicates whether one goroutine is running netcheck
	_refreshingReportLock heavylock.CASMutex
)

func init() {
	// _needToReport and _forceToReportOnce are automatically statically
	// initialized with zero-value of atomicutil.AtomicBoolean type.

	var nilCheckReportPtr *CheckReport = nil
	_neverDirectRW_atomic_lastReportPtr.Store(nilCheckReportPtr)

	_refreshingReportLock = heavylock.NewCASMutex()
}

// RequestNetcheck would asynchronously invoke netcheck program for network
// diagnostic, when no other network diagnostic is running or the last
// diagnostic report has outdated.
func RequestNetcheck(requestType NetcheckRequestType) {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"module": "checknet",
	})

	switch requestType {
	case NetcheckRequestNormal:
		_needToReport.Set()
		wrapgo.GoWithDefaultPanicHandler(func() {
			_doNetcheck(NetcheckRequestNormal)
		})
	case NetcheckRequestForceOnce:
		wrapgo.GoWithDefaultPanicHandler(func() {
			_doNetcheck(NetcheckRequestForceOnce)
		})
	default:
		logger.WithFields(logrus.Fields{
			"requestType": requestType,
		}).Errorln("Invalid netcheck request type")
		return
	}
}

func _doNetcheck(requestType NetcheckRequestType) {
	if !_refreshingReportLock.TryLock() {
		return
	}
	defer _refreshingReportLock.Unlock()

	logger := log.GetLogger().WithFields(logrus.Fields{
		"module": "checknet",
	})
	// Only check cache validity when processing normal netcheck request
	if requestType == NetcheckRequestNormal {
		reportPtr, ok := _neverDirectRW_atomic_lastReportPtr.Load().(*CheckReport)
		if !ok {
			return
		}
		if reportPtr != nil {
			if !isReportOutdated(reportPtr.FinishedTime) {
				return
			}
		}
	}

	logger.WithFields(logrus.Fields{
		"requestType": requestType,
	}).Infoln("Invoke netcheck program in response to checknet request")
	resultCode, err := invokeNetcheck()
	if err != nil {
		logger.WithError(err).Errorln("Failed to invoke netcheck program")
		return
	}

	finishedTime := time.Now()
	newReportPtr := &CheckReport{
		Result:       resultCode,
		FinishedTime: finishedTime,
	}
	_neverDirectRW_atomic_lastReportPtr.Store(newReportPtr)
	// Only increase force-to-report response counter when processing
	// force-to-report netcheck request
	if requestType == NetcheckRequestForceOnce {
		_forceToReportResponses.Add(1)
	}
	logger.WithFields(logrus.Fields{
		"result":       resultCode,
		"finishedTime": finishedTime.Format(time.RFC3339),
	}).Infoln("Finished network diagnostic")
}

// RecentReport would return the most recent available network diagnostic report,
// or nil pointer if the report has not been generated. When the report has been
// outdated, it would call RequestNetcheck to refresh netcheck report.
func RecentReport() *CheckReport {
	isForceToReportOnce := _forceToReportResponses.Load() > 0
	isNeedToReport := _needToReport.IsSet()
	if !isForceToReportOnce && !isNeedToReport {
		return nil
	}

	reportPtr, ok := _neverDirectRW_atomic_lastReportPtr.Load().(*CheckReport)
	if !ok {
		return nil
	}
	if reportPtr == nil {
		return nil
	}

	// NOTE: Thanks to to serial feature of gshell channel, RecentReport() would
	// never be called concurrently and _forceToReportResponses counter should
	// never become less than zero due to parallel decreasing actions more than
	// available response count.
	if isForceToReportOnce {
		_forceToReportResponses.Add(-1)
	}
	// Only when it is needed to report by automatic detection via heart-beat,
	// it is needed to check whether current report is out-of-date.
	if isNeedToReport && isReportOutdated(reportPtr.FinishedTime) {
		RequestNetcheck(NetcheckRequestNormal)
	}

	return reportPtr
}

// DeclareNetworkCategory sets the network category in cache of this module,
// which is used to specify the network environment when running netcheck
// program.
func DeclareNetworkCategory(category networkcategory.NetworkCategory) {
	networkCategoryCache.Set(category)
}

// clearNeedToReport simply set that it is not needed to report network
// diagnostic result.
func clearNeedToReport() {
	_needToReport.Clear()
}
