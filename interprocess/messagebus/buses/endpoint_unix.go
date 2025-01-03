//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package buses

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
)

const (
	centralUDSName = "aliyun_assist_service.sock"
)

func GetCentralEndpoint(overWrite bool) Endpoint {
	udsPath := filepath.Join(os.TempDir(), centralUDSName)
	if overWrite && fileutil.CheckFileIsExist(udsPath) {
		os.Remove(udsPath)
	}

	return Endpoint{
		protocol: UnixDomainSocketProtocol,
		path:     udsPath,
	}
}

func (e *Endpoint) Parse(endpoint string) error {
	items := strings.Split(endpoint, "://")
	if len(items) != 2 {
		return fmt.Errorf("unknown endpoint")
	}
	e.protocol = items[0]
	e.path = items[1]
	return nil
}

func (e *Endpoint) String() string      { return fmt.Sprintf("%s://%s", e.protocol, e.path) }
