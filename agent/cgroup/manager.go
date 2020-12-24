//+build linux

package cgroup

import (
	"errors"
	"fmt"
)

var (
	ErrCgroupRemoved = errors.New("Unable to continue. Control group already removed")
)

// The wrapper around the several control groups to more convenient using.
type Manager struct {
	pid       int
	cgroups   map[string]Cgroup
	isRemoved bool
}

func NewManager(pid int, subpath string, subsystems ...string) (*Manager, error) {
	cgroupList, err := GetEnabledSubsystems()
	if err != nil {
		return nil, err
	}

	for _, s := range subsystems {
		if _, ok := cgroupList[s]; !ok {
			return nil, fmt.Errorf("Unknown subsystem: %s", s)
		}
	}

	cgroups := make(map[string]Cgroup)

	for _, s := range subsystems {
		switch s {
		case "cpu":
			g, err := NewCpuGroup(subpath, pid)
			if err != nil {
				return nil, NewCgroupInitError(s, err)
			}
			cgroups[s] = g
		case "memory":
			g, err := NewMemoryGroup(subpath, pid)
			if err != nil {
				return nil, NewCgroupInitError(s, err)
			}
			cgroups[s] = g
		default:
			return nil, NewUnsupportedError(s)
		}
	}

	return &Manager{pid: pid, cgroups: cgroups}, nil
}

func LoadManager(pid int) (*Manager, error) {
	cgroupList, err := GetProcessCgroups(pid)
	if err != nil {
		return nil, err
	}

	if len(cgroupList) == 0 {
		return nil, NewCgroupsNotFoundError(pid)
	}

	cgroups := make(map[string]Cgroup)

	for s, _ := range cgroupList {
		switch g, err := LookupCgroupByPid(pid, s); {
		case err == nil:
			cgroups[s] = g
		case IsUnsupportedError(err):
			continue
		default:
			return nil, err
		}
	}

	return &Manager{pid: pid, cgroups: cgroups}, nil
}

func (m *Manager) GetPid() int {
	return m.pid
}

func (m *Manager) Set(c *Config) error {
	if m.isRemoved {
		return ErrCgroupRemoved
	}

	for _, g := range m.cgroups {
		if err := g.Set(c); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Get(c *Config) error {
	if m.isRemoved {
		return ErrCgroupRemoved
	}

	for _, g := range m.cgroups {
		if err := g.Get(c); err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) Destroy() error {
	m.isRemoved = true

	for _, g := range m.cgroups {
		if err := DestroyCgroup(g.GetPath()); err != nil {
			return err
		}
	}

	return nil
}

type CgroupInitError struct {
	Subsystem string
	Err       error
}

func (e *CgroupInitError) Error() string {
	return fmt.Sprintf("Cannot create %s control group: %s", e.Subsystem, e.Err)
}

func NewCgroupInitError(subsystem string, err error) error {
	return &CgroupInitError{subsystem, err}
}

func IsCgroupInitError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*CgroupInitError)
	return ok
}

type CgroupsNotFoundError struct {
	pid int
}

func (e *CgroupsNotFoundError) Error() string {
	return fmt.Sprintf("No one control group is found: %d", e.pid)
}

func NewCgroupsNotFoundError(pid int) error {
	return &CgroupsNotFoundError{pid}
}

func IsCgroupsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*CgroupsNotFoundError)
	return ok
}
