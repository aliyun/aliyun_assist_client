//go:build windows

package processcollection

import (
	"errors"
	"fmt"

	"github.com/shirou/gopsutil/v3/process"
)

func (g *ProcessTree) KillAll() error {
	for _, root := range g.processList {
		p, err := process.NewProcess(int32(root.Pid))
		if err != nil {
			return fmt.Errorf("find process %d failed: %v", root.Pid, err)
		}
		if err := killRecur(p); err != nil {
			return err
		}
	}
	return nil
}

func killRecur(rootProcess *process.Process) error {
	children, err := rootProcess.Children()
	if err != nil && !errors.Is(err, process.ErrorNoChildren) {
		return fmt.Errorf("find children of process %d failed: %v", rootProcess.Pid, err)
	}
	for _, p := range children {
		if err := killRecur(p); err != nil {
			return err
		}
	}
	if err := rootProcess.Kill(); err != nil && !errors.Is(err, process.ErrorProcessNotRunning) {
		cmdline, cmdlineErr := rootProcess.Cmdline()
		if cmdlineErr != nil {
			cmdline = "unknown-process"
		}
		return fmt.Errorf("Failed to kill main or descendant process %d(%s): %s", rootProcess.Pid, cmdline, err.Error())
	}
	return nil
}
