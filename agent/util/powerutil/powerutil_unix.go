//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package powerutil

const (
	powerdownCmd = "shutdown -h now"
	rebootCmd = "shutdown -r now"
)
