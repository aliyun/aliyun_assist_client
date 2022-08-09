package metrics

import(
	"bytes"
	"golang.org/x/sys/unix"
)

func getKernelVersion() string {
	var utsn unix.Utsname
	err := unix.Uname(&utsn)
	if err != nil {
		return ""
	}
	releaseLength := bytes.IndexByte(utsn.Release[:], '\u0000')
	return string(utsn.Release[:releaseLength])
}