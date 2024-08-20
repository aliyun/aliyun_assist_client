//go:build !windows

package processcollection

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/shirou/gopsutil/v3/process"
)

func (g *ProcessTree) KillAll() error {
	childPidMap, cmdlineMap, err := probeProcessMap()
	if err != nil {
		return fmt.Errorf("Probe process failed: %v", err)
	}
	for _, root := range g.processList {
		if err := killRecur(int32(root.Pid), childPidMap, cmdlineMap); err != nil {
			return err
		}
	}
	return nil
}

func probeProcessMap() (map[int32][]int32, map[int32]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()

	allProcesses, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("probe all processes failed, %v", err)
	}
	childPidMap := make(map[int32][]int32)
	cmdlineMap := make(map[int32]string)
	for _, p := range allProcesses {
		ppid, err := p.PpidWithContext(ctx)
		if err != nil {
			return nil, nil, fmt.Errorf("get ppid of pid[%d] failed,  %v", p.Pid, err)
		}
		processName, _ := p.CmdlineWithContext(ctx)
		if processName == "" {
			processName, err = p.NameWithContext(ctx)
			if err != nil {
				log.GetLogger().Infof("get name of pid[%d] failed, %v",p.Pid, err)
				processName = "unknown-process"
			}
		}
		childPidMap[ppid] = append(childPidMap[ppid], p.Pid)
		cmdlineMap[p.Pid] = processName
	}
	return childPidMap, cmdlineMap, nil
}

func killRecur(rootPid int32, childPidMap map[int32][]int32, cmdlineMap map[int32]string) error {
	if children, ok := childPidMap[rootPid]; ok {
		for _, pid := range children {
			if err := killRecur(pid, childPidMap, cmdlineMap); err != nil {
				return err
			}
		}
	}
	if err := syscall.Kill(int(rootPid), syscall.SIGKILL); err != nil {
		var processName string
		if name, ok := cmdlineMap[rootPid]; ok {
			processName = name
		} else {
			processName = "unknown"
		}
		return fmt.Errorf("Failed to kill main or descendant process %d(%s): %s", rootPid, processName, err.Error())
	}
	return nil
}
