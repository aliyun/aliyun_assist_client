// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package update

func categorizeExitCode(exitCode int) string {
	if exitCode == 0 {
		return ""
	} else if exitCode == 127 {
		return ":FileNotExist"
	} else if exitCode > 128 {
		return ":ExitedBySignal"
	} else if exitCode == -1 {
		return ":Killed"
	} else {
		return ":UnexpectedExitStatus"
	}
}
