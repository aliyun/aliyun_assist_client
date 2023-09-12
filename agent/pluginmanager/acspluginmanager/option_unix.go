//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package acspluginmanager

type ExecuteParams struct {
	CommonExecuteParams

	Foreground bool
}
