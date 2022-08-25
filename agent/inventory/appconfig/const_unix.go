//go:build darwin || freebsd || linux || netbsd || openbsd
// +build darwin freebsd linux netbsd openbsd

package appconfig

import (
	"path/filepath"

	libupdate "github.com/aliyun/aliyun_assist_client/common/update"
)

var DefaultDataStorePath string = filepath.Join(libupdate.DefaultUnixInstallDir, "data")
