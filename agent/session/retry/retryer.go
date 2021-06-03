package retry

import (
	"math"
	"math/rand"
	"time"
)

type Retryer interface {
	Call() error
	NextSleepTime(int32) time.Duration
}

// TODO Move to a common package for retry and merge with HibernateRetryStrategy
type ExponentialRetryer struct {
	CallableFunc   func() (interface{}, error)
	GeometricRatio float64
	// a random amount of jitter up to JitterRatio percentage, 0.0 means no jitter, 0.15 means 15% added to the total wait time.
	JitterRatio         float64
	InitialDelayInMilli int
	MaxDelayInMilli     int
	MaxAttempts         int
}

// Init initializes the retryer
func (retryer *ExponentialRetryer) Init() {
	rand.Seed(time.Now().UnixNano())
}

// NextSleepTime calculates the next delay of retry. Returns next sleep time as well as if it reaches max delay
func (retryer *ExponentialRetryer) NextSleepTime(attempt int) (time.Duration, bool) {
	sleep := time.Duration(float64(retryer.InitialDelayInMilli)*math.Pow(retryer.GeometricRatio, float64(attempt))) * time.Millisecond
	exceedMaxDelay := false
	if int(sleep/time.Millisecond) > retryer.MaxDelayInMilli {
		sleep = time.Duration(retryer.MaxDelayInMilli) * time.Millisecond
		exceedMaxDelay = true
	}
	jitter := int64(0)
	maxJitter := int64(float64(sleep) * retryer.JitterRatio)
	if maxJitter > 0 {
		jitter = rand.Int63n(maxJitter)
	}
	return sleep + time.Duration(jitter), exceedMaxDelay
}

// Call calls the operation and does exponential retry if error happens until it reaches MaxAttempts if specified.
func (retryer *ExponentialRetryer) Call() (channel interface{}, err error) {
	attempt := 0
	failedAttemptsSoFar := 0
	for {
		channel, err := retryer.CallableFunc()
		if err == nil || failedAttemptsSoFar == retryer.MaxAttempts {
			return channel, err
		}
		sleep, exceedMaxDelay := retryer.NextSleepTime(attempt)
		if !exceedMaxDelay {
			attempt++
		}
		time.Sleep(sleep)
		failedAttemptsSoFar++
	}
}
