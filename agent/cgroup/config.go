// +build linux

package cgroup

type Config struct {
	// CPU 使用率权重
	CpuShares int64 `json:"cpu_shares"`

	// 以下两个参数用于限制CPU最高使用率
	CpuQuota  int64 `json:"cpu_quota"`
	CpuPeriod int64 `json:"cpu_period"`

	// 以下两个参数用于设置CPU最高使用率
	CpuRtRuntime int64 `json:"cpu_rt_runtime"`
	CpuRtPeriod  int64 `json:"cpu_rt_period"`

	//限制最大内存使用量
	MemoryLimit int64 `json:"memory_quota"`
}
