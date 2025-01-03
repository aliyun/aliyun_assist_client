package idlecheck

import (
	"sync"
	"time"
)

const (
	minExtendSecond = 5
)

type checker func() bool

var (
	extendSecond = minExtendSecond

	whenExit    time.Time
	checkerList map[string]checker
	l           sync.Mutex
)

func ExtendLive() {
	l.Lock()
	defer l.Unlock()

	when := time.Now().Add(time.Second * time.Duration(extendSecond))
	if when.After(whenExit) {
		whenExit = when
	}
}

func TimeToExit() (time.Time, bool) {
	l.Lock()
	defer l.Unlock()

	for _, f := range checkerList {
		if couldExit := f(); !couldExit {
			whenExit = time.Now().Add(time.Second * time.Duration(extendSecond))
			return whenExit, false
		}
	}
	if time.Now().After(whenExit) {
		return whenExit, true
	}
	return whenExit, false
}

func SetChecker(name string, f checker) {
	l.Lock()
	defer l.Unlock()

	if checkerList == nil {
		checkerList = map[string]checker{}
	}
	checkerList[name] = f
}

func SetExtendSecond(waittime int) {
	l.Lock()
	defer l.Unlock()

	if waittime < minExtendSecond {
		waittime = minExtendSecond
	}
	extendSecond = waittime
}
