package endpoint

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
)

var (
	socketName string
	protocol   string
)

var (
	ep *buses.Endpoint
	l  sync.Mutex
)

func SetSocketAndProtocol(socketStr string, protocolStr string) {
	socketName = socketStr
	protocol = protocolStr
}

func GetEndpoint(overWrite bool) *buses.Endpoint {
	l.Lock()
	defer l.Unlock()

	if ep == nil {
		var udsPath string
		if protocol == "unix" {
			udsPath = filepath.Join(os.TempDir(), socketName)
		} else if protocol == "npipe" {
			udsPath = fmt.Sprintf(`\\.\pipe\%s`, socketName)
		} else {
			panic(fmt.Sprintf("unsupported protocol: %s", protocol))
		}

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
