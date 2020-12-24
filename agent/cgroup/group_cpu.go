//+build linux

package cgroup

import (
	"errors"
	"os"
	"strconv"
)

var (
	ErrCfsNotEnabled = errors.New("Make sure that CONFIG_CFS_BANDWIDTH option is enabled in your kernel")
	ErrRtNotEnabled  = errors.New("Make sure that CONFIG_RT_GROUP_SCHED option is enabled in your kernel")
)

type CpuGroup struct {
	path string
}

func NewCpuGroup(subpath string, pid int) (Cgroup, error) {
	return NewGroup(subpath, "cpu", pid)
}

func (g *CpuGroup) Set(c *Config) error {
	if c.CpuShares != 0 {
		if err := writeValue(g.path, "cpu.shares", strconv.FormatInt(c.CpuShares, 10)); err != nil {
			return err
		}
	}
	if c.CpuPeriod != 0 {
		switch err := writeValue(g.path, "cpu.cfs_period_us", strconv.FormatInt(c.CpuPeriod, 10)); {
		case err == nil:
		case os.IsNotExist(err):
			return ErrCfsNotEnabled
		default:
			return err
		}
	}
	if c.CpuQuota != 0 {
		switch err := writeValue(g.path, "cpu.cfs_quota_us", strconv.FormatInt(c.CpuQuota, 10)); {
		case err == nil:
		case os.IsNotExist(err):
			return ErrCfsNotEnabled
		default:
			return err
		}
	}
	if c.CpuRtPeriod != 0 {
		switch err := writeValue(g.path, "cpu.rt_period_us", strconv.FormatInt(c.CpuRtPeriod, 10)); {
		case err == nil:
		case os.IsNotExist(err):
			return ErrRtNotEnabled
		default:
			return err
		}
	}
	if c.CpuRtRuntime != 0 {
		switch err := writeValue(g.path, "cpu.rt_runtime_us", strconv.FormatInt(c.CpuRtRuntime, 10)); {
		case err == nil:
		case os.IsNotExist(err):
			return ErrRtNotEnabled
		default:
			return err
		}
	}

	return nil
}

func (g *CpuGroup) Get(c *Config) error {
	switch v, err := readInt64Value(g.path, "cpu.shares"); {
	case err == nil:
		c.CpuShares = v
	default:
		return err
	}

	switch v, err := readInt64Value(g.path, "cpu.cfs_period_us"); {
	case err == nil:
		c.CpuPeriod = v
	case os.IsNotExist(err):
	default:
		return err
	}

	switch v, err := readInt64Value(g.path, "cpu.cfs_quota_us"); {
	case err == nil:
		c.CpuQuota = v
	case os.IsNotExist(err):
	default:
		return err
	}

	switch v, err := readInt64Value(g.path, "cpu.rt_period_us"); {
	case err == nil:
		c.CpuRtPeriod = v
	case os.IsNotExist(err):
	default:
		return err
	}

	switch v, err := readInt64Value(g.path, "cpu.cfs_runtime_us"); {
	case err == nil:
		c.CpuRtRuntime = v
	case os.IsNotExist(err):
	default:
		return err
	}

	return nil
}

func (g *CpuGroup) GetPath() string {
	return g.path
}
