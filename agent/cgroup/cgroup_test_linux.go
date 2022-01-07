package cgroup

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCgroup(t *testing.T) {
	var err error
	cgroup_name := "test_cgroup"
	manager, err := NewManager(os.Getpid(), cgroup_name, "cpu", "memory")
	assert.Equal(t, nil, err)
	manager.GetPid()
	cfg := &Config{
		CpuQuota: int64(1000),
	}
	err = manager.Set(cfg)
	assert.Equal(t, nil, err)

	err = manager.Get(cfg)
	assert.Equal(t, nil, err)
	assert.NotEqual(t, nil, cfg)

	LoadManager(os.Getpid())
	manager.Destroy()
}