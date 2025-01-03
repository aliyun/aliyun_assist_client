package util

import (
	"strconv"
	"sync"
	"time"
)

type Mtoken struct {
	token           string
	expireTimestamp time.Time
	l               sync.Mutex
}

const (
	tokenTimeoutSecond = 60
)

var mt Mtoken

func MetaserverToken() (string, error) {
	mt.l.Lock()
	defer mt.l.Unlock()

	if time.Now().Before(mt.expireTimestamp) {
		return mt.token, nil
	}
	token, err := HttpPutWithHeaders("http://100.100.100.200/latest/api/token", map[string]string{
		"X-aliyun-ecs-metadata-token-ttl-seconds": strconv.Itoa(tokenTimeoutSecond + 5),
	})
	if err != nil {
		return "", err
	}
	mt.expireTimestamp = time.Now().Add(time.Second * time.Duration(tokenTimeoutSecond))
	mt.token = token
	return mt.token, nil
}
