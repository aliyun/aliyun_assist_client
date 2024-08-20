//go:build linux
// +build linux

package osutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/systemdutil"
)

func IsSystemShutdown(ctx context.Context) (bool, error) {
	var (
		res         bool
		systemdErr  error
		runlevelErr error
	)
	if systemdutil.IsRunningSystemd() {
		if res, systemdErr = checkShutdownBySystemstate(ctx); systemdErr == nil {
			return res, nil
		}
	}
	if res, runlevelErr = checkShutdownByRunlevel(ctx); runlevelErr == nil {
		if res {
			return res, nil
		}
	}
	return false, fmt.Errorf("systemdErr: %v, runlevelErr: %v", systemdErr, runlevelErr)
}

// Use runlevel to determine whether the os is shutting down, it is not reliable.
// True indicates that the os is restarting or shutting down, but it does not
// always return true when the os is restarting or shutting down.
func checkShutdownByRunlevel(ctx context.Context) (bool, error) {
	output, err := exec.CommandContext(ctx, "runlevel").CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("runlevel error, %s, %v", string(output), err)
	}
	log.GetLogger().Info("checkShutdownByRunlevel: ", string(output))
	fields := strings.Split(strings.TrimSpace(string(output)), " ")
	if len(fields) != 2 {
		return false, fmt.Errorf("unknown runlevel")
	}
	switch fields[1] {
	case "0":
		return true, nil
	case "6":
		return true, nil
	case "N":
		return false, fmt.Errorf("unknown runlevel")
	default:
		return false, nil
	}
}

// Use system state of systemd to determine whether the os is shutting down. 
// It is not reliable too.
func checkShutdownBySystemstate(ctx context.Context) (bool, error) {
	systemState, err := systemdutil.SystemState(ctx)
	if err != nil {
		return false, fmt.Errorf("system state error, %v", err)
	}
	log.GetLogger().Info("checkShutdownBySystemstate: ", systemState)
	return systemState == "stopping", nil
}
