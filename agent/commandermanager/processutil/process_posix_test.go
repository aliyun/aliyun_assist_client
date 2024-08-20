//go:build !windows
// +build !windows

package processutil

var (
	sleepCommand = []string{"sh", "-c", "sleep 5"}
)