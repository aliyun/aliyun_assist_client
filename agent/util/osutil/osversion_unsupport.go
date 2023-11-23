// +build !windows,!linux,!freebsd

package osutil

func getVersion() string {
	return ""
}

func GetUnameMachine() (string, error) {
	return "", nil
}

func getKernelVersion() string {
	return ""
}