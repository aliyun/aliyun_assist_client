package endpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

const (
	// default agent endpoint
	sockerName = "aliyun_assist_container.sock"
	protocol   = "unix"
)

var (
	ep *buses.Endpoint
	l  sync.Mutex
)

func GetEndpoint(overWrite bool) *buses.Endpoint {
	l.Lock()
	defer l.Unlock()

	if ep == nil {
		udsPath := filepath.Join(os.TempDir(), sockerName)
		ep = &buses.Endpoint{}
		ep.Parse(fmt.Sprintf("%s://%s", protocol, udsPath))
	}

	if overWrite && fileutil.CheckFileIsExist(ep.GetPath()) {
		os.Remove(ep.GetPath())
	}

	return ep
}

func SetEndpoint(e *buses.Endpoint) {
	l.Lock()
	defer l.Unlock()

	ep = e
}
