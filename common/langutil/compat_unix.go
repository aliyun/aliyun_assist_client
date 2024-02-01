//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package langutil

func LocalToUTF8(local string) string {
	return local
}

func UTF8ToLocal(utf8String string) string {
	return utf8String
}
