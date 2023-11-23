package envutil

import (
	"fmt"
	"os"
	"strings"
)

func ClearExecErrDot() {
	goDebug := os.Getenv("GODEBUG")
	if strings.Contains(goDebug, "execerrdot=0") {
		return
	}
	if len(goDebug) == 0 {
		goDebug = "execerrdot=0"
	} else {
		goDebug = fmt.Sprintf("execerrdot=0,%s", goDebug)
	}
	os.Setenv("GODEBUG", goDebug)
}