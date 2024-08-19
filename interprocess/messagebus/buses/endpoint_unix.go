//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package buses

import (
	"os"
	"path/filepath"

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
		path: udsPath,
	}
}
