// +build windows

package update

import (
	"strings"
)

func isNoEnoughSpaceError(err error) bool {
	// TODO: Replace error message string matching with error type or attribute
	// assertion. Currently code for no enough space error under Windows is not
	// found.
	return strings.Contains(err.Error(), "There is not enough space on the disk.")
}

func categorizeExitCode(exitCode int) string {
	// TODO: Deep into Windows error code and provide better categories
	return ""
}
