//go:build !linux
// +build !linux

package systemdutil

func IsRunningSystemd() bool {
	return false
}