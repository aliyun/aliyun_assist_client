// +build linux

package cgroup

type Stats struct {
	CpuStats CpuStats `json:"cpu_stats,omitempty"`
}

type CpuUsage struct {
	// Total CPU time consumed (in nanoseconds).
	TotalUsage uint64 `json:"total_usage,omitempty"`

	// Total CPU time consumed per core (in nanoseconds).
	PercpuUsage []uint64 `json:"percpu_usage,omitempty"`

	// Time spent by tasks of the cgroup in kernel mode (in nanoseconds).
	UsageInKernelmode uint64 `json:"usage_in_kernelmode"`

	// Time spent by tasks of the cgroup in user mode (in nanoseconds).
	UsageInUsermode uint64 `json:"usage_in_usermode"`
}

type ThrottlingData struct {
	// Number of periods with throttling active.
	Periods uint64 `json:"nr_periods,omitempty"`

	// Number of times when the tasks have been throttled.
	ThrottledPeriods uint64 `json:"nr_throttled,omitempty"`

	// Aggregate time when the tasks have been throttled (in nanoseconds).
	ThrottledTime uint64 `json:"throttled_time,omitempty"`
}

type CpuStats struct {
	CpuUsage       CpuUsage       `json:"cpu_usage,omitempty"`
	ThrottlingData ThrottlingData `json:"throttling_data,omitempty"`
}
