//+build linux

package cgroup

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Cgroup interface {
	Set(*Config) error

	Get(*Config) error

	GetPath() string
}

func NewGroup(subpath string, subsystem string, pid int) (Cgroup, error) {
	subsystemPath, err := GetSubsystemMountpoint(subsystem)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(subsystemPath, subpath)

	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}

	if err := writeValue(path, "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return nil, err
	}
	if subsystem == "cpu" {
		return Cgroup(&CpuGroup{path}), nil
	} else if subsystem == "memory" {
		return Cgroup(&MemoryGroup{path}), nil
	} else {
		return nil, errors.New("Invalid subsystem")
	}
}

func LookupCgroupByPid(pid int, subsystem string) (Cgroup, error) {
	subsystemPath, err := GetSubsystemMountpoint(subsystem)
	if err != nil {
		return nil, err
	}

	subpath, err := GetCgroupPathByPid(pid, subsystem)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(subsystemPath, subpath)

	var g Cgroup

	switch subsystem {
	case "cpu":
		g = Cgroup(&CpuGroup{path})
	case "memory":
		g = Cgroup(&MemoryGroup{path})
	default:
		return nil, NewUnsupportedError(subsystem)
	}

	return g, nil
}

func DestroyCgroup(path string) error {
	os.RemoveAll(path)

	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func GetEnabledSubsystems() (map[string]int, error) {
	cgroupsFile, err := os.Open("/proc/cgroups")
	if err != nil {
		return nil, err
	}
	defer cgroupsFile.Close()

	scanner := bufio.NewScanner(cgroupsFile)

	// Skip the first line. It's a comment
	scanner.Scan()

	cgroups := make(map[string]int)
	for scanner.Scan() {
		var subsystem string
		var hierarchy int
		var num int
		var enabled int
		fmt.Sscanf(scanner.Text(), "%s %d %d %d", &subsystem, &hierarchy, &num, &enabled)

		if enabled == 1 {
			cgroups[subsystem] = hierarchy
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Cannot parsing /proc/cgroups: %s", err)
	}

	return cgroups, nil
}

func GetSubsystemMountpoint(subsystem string) (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				return fields[4], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("Mountpoint not found: %s", subsystem)
}

func GetCgroupPathByPid(pid int, subsystem string) (string, error) {
	cgroups, err := GetProcessCgroups(pid)
	if err != nil {
		return "", err
	}

	for s, p := range cgroups {
		if s == subsystem {
			return p, nil
		}
	}

	return "", fmt.Errorf("Not in subsystem %s: %d", subsystem, pid)
}

func GetProcessCgroups(pid int) (map[string]string, error) {
	fname := fmt.Sprintf("/proc/%d/cgroup", pid)

	cgroups := make(map[string]string)

	f, err := os.Open(fname)
	if err != nil {
		return nil, fmt.Errorf("Cannot open %s: %s", fname, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 3)
		if len(parts) < 3 {
			return nil, fmt.Errorf("Cannot parsing %s: unknown format", fname)
		}
		subsystemsParts := strings.Split(parts[1], ",")
		for _, s := range subsystemsParts {
			cgroups[s] = parts[2]
		}
	}

	return cgroups, nil
}

type UnsupportedError struct {
	Subsystem string
}

func (e *UnsupportedError) Error() string {
	return fmt.Sprintf("Unsupported subsystem: %s", e.Subsystem)
}

func NewUnsupportedError(subsystem string) error {
	return &UnsupportedError{subsystem}
}

func IsUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*UnsupportedError)
	return ok
}
