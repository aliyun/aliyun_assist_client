// +build windows

package appconfig

import (
	"path/filepath"

	libupdate "github.com/aliyun/aliyun_assist_client/common/update"
)

var DefaultDataStorePath string = filepath.Join(libupdate.DefaultWindowsInstallDir, "data")
