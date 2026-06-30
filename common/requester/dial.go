package requester

import (
	"net"
	"time"
)

var (
	defaultDialContextFunc = (&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		DualStack: true,
	}).DialContext
)
