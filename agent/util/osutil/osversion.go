// +build !windows

package osutil

import (
	"bytes"
	"fmt"

	"golang.org/x/sys/unix"
)

// GetVersion returns OS version string depends on different types of OS
func GetVersion() string {
	// The linux version concatenated string from Utsname struct, which mimics
	// original C++ agent result
	var utsn unix.Utsname
	err := unix.Uname(&utsn)
	if err != nil {
		return ""
	}

	// utsn.Sysname/Version/Machine fields are all [256]byte array, which hold
	// zero-terminated C-style string and need extra step to determine real length
	// and convert meaningful part to string
	sysnameLength := bytes.IndexByte(utsn.Sysname[:], 0)
	sysname := string(utsn.Sysname[:sysnameLength])
	versionLength := bytes.IndexByte(utsn.Version[:], 0)
	version := string(utsn.Version[:versionLength])
	machineLength := bytes.IndexByte(utsn.Machine[:], 0)
	machine := string(utsn.Machine[:machineLength])
	return fmt.Sprintf("%s_%s_%s", sysname, version, machine)
}

func GetUnameMachine() (string, error) {
	var utsn unix.Utsname
	err := unix.Uname(&utsn)
	if err != nil {
		return "", err
	}

	machineLength := bytes.IndexByte(utsn.Machine[:], 0)
	machine := string(utsn.Machine[:machineLength])
	return machine, nil
}
