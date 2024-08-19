//go:build linux
// +build linux

package systemdutil

import (
	"context"
	"os"
	"strings"
	"sync"

	systemdDbus "github.com/coreos/go-systemd/v22/dbus"
)

var (
	isRunningSystemdOnce sync.Once
	isRunningSystemd     bool
)

// NOTE: This function comes from package github.com/coreos/go-systemd/util
// It was borrowed here to avoid a dependency on cgo.
//
// IsRunningSystemd checks whether the host was booted with systemd as its init
// system. This functions similarly to systemd's `sd_booted(3)`: internally, it
// checks whether /run/systemd/system/ exists and is a directory.
// http://www.freedesktop.org/software/systemd/man/sd_booted.html
func IsRunningSystemd() bool {
	isRunningSystemdOnce.Do(func() {
		fi, err := os.Lstat("/run/systemd/system")
		isRunningSystemd = err == nil && fi.IsDir()
	})
	return isRunningSystemd
}

func SystemState(ctx context.Context) (string, error) {
	cm := newDbusConnManager()
	var property *systemdDbus.Property
	var err error
	cm.retryOnDisconnect(func(c *systemdDbus.Conn) error {
		property, err = c.SystemStateContext(ctx)
		return err
	})
	if err != nil {
		return "", err
	}
	return strings.Trim(property.Value.String(), "\""), nil
}