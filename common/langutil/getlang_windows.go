package langutil

import (
	"golang.org/x/sys/windows"
)

var (
	Modkernel32           = windows.NewLazySystemDLL("kernel32.dll")
	pGetSystemDefaultLCID = Modkernel32.NewProc("GetSystemDefaultLCID")
)

func GetDefaultLang() uint32 {
	r, _, _ := pGetSystemDefaultLCID.Call()
	return uint32(r)
}
