//+build linux

package cgroup

import (
	"strconv"
)

type MemoryGroup struct {
	path string
}

func NewMemoryGroup(subpath string, pid int) (Cgroup, error) {
	return NewGroup(subpath, "memory", pid)
}

func (g *MemoryGroup) Set(c *Config) error {
	if c.MemoryLimit != 0 {
		//纯物理内存限制
		if err := writeValue(g.path, "memory.limit_in_bytes", strconv.FormatInt(c.MemoryLimit, 10)); err != nil {
			return err
		}
		//物理内存+交换文件限制
		if err := writeValue(g.path, "memory.memsw.limit_in_bytes", strconv.FormatInt(c.MemoryLimit*2, 10)); err != nil {
			return err
		}
	}
	return nil
}

func (g *MemoryGroup) Get(c *Config) error {
	switch v, err := readInt64Value(g.path, "memory.limit_in_bytes"); {
	case err == nil:
		c.MemoryLimit = v
	default:
		return err
	}
	return nil
}

func (g *MemoryGroup) GetPath() string {
	return g.path
}
