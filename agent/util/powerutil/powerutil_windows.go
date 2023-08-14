//go:build windows
// +build windows

package powerutil

const (
	powerdownCmd = "shutdown -f -s -t 0"
	rebootCmd = "shutdown -f -r -t 0"
)
