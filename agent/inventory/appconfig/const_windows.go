// +build windows

package appconfig

import (
	"path/filepath"

	"github.com/aliyun/aliyun_assist_client/agent/update"
)

var DefaultDataStorePath string = filepath.Join(update.DefaultWindowsInstallDir, "data")
