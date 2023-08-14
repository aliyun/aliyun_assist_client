// +build windows

package acspluginmanager

import (
	"fmt"
)

func errProcess(function string, exitCode int, err error, tip string) (int, string) {
	var errorCode string
	var ok bool
	if errorCode, ok = ErrorStrMap[exitCode]; !ok {
		errorCode = "UNKNOWN"
	}
	fmt.Printf("%s %s: %s\n", function, errorCode, tip)
	return exitCode, errorCode
}