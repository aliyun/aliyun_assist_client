// + build windows

package instancedetailedinformation

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/stretchr/testify/assert"
)

var (
	sampleDataWindows = [][]string{
		{
			// Windows Server 2016 c4.8xlarge
			`{"CPUModel":"Intel(R) Xeon(R) CPU E5-2666 v3 @ 2.90GHz","CPUSpeedMHz":"2900","CPUs":"36","CPUSockets":"2","CPUCores":"18","CPUHyperThreadEnabled":"true"}`,
			`{"OSServicePack":"0"}`,
		},
		{
			// Windows Server 2016 t2.2xlarge
			`{"CPUModel":"Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz","CPUSpeedMHz":"2395","CPUs":"8","CPUSockets":"1","CPUCores":"8","CPUHyperThreadEnabled":"false"}`,
			`{"OSServicePack":"0"}`,
		},
		{
			// Windows Server 2003 R2 t2.2xlarge
			`{"CPUModel":"Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz","CPUSpeedMHz":"2395","CPUs":"8","CPUSockets":"8","CPUCores":"8","CPUHyperThreadEnabled":"false"}`,
			`{"OSServicePack":"2"}`,
		},
		{
			// Windows Server 2008 R2 SP1 m4.16xlarge
			`{"CPUModel":"Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz","CPUSpeedMHz":"2301","CPUs":"64","CPUSockets":"2","CPUCores":"32","CPUHyperThreadEnabled":"true"}`,
			`{"OSServicePack":"1"}`,
		},
	}
)

var sampleDataWindowsParsed = []model.InstanceDetailedInformation{
	{
		CPUModel:              "Intel(R) Xeon(R) CPU E5-2666 v3 @ 2.90GHz",
		CPUSpeedMHz:           "2900",
		CPUs:                  "36",
		CPUSockets:            "2",
		CPUCores:              "18",
		CPUHyperThreadEnabled: "true",
		OSServicePack:         "0",
	},
	{
		CPUModel:              "Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz",
		CPUSpeedMHz:           "2395",
		CPUs:                  "8",
		CPUSockets:            "1",
		CPUCores:              "8",
		CPUHyperThreadEnabled: "false",
		OSServicePack:         "0",
	},
	{
		CPUModel:              "Intel(R) Xeon(R) CPU E5-2676 v3 @ 2.40GHz",
		CPUSpeedMHz:           "2395",
		CPUs:                  "8",
		CPUSockets:            "8",
		CPUCores:              "8",
		CPUHyperThreadEnabled: "false",
		OSServicePack:         "2",
	},
	{
		CPUModel:              "Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz",
		CPUSpeedMHz:           "2301",
		CPUs:                  "64",
		CPUSockets:            "2",
		CPUCores:              "32",
		CPUHyperThreadEnabled: "true",
		OSServicePack:         "1",
	},
}

func TestCollectPlatformDependentInstanceData(t *testing.T) {
	for i, sampleCPUAndOSData := range sampleDataWindows {
		sampleCPUData, sampleOSData := sampleCPUAndOSData[0], sampleCPUAndOSData[1]
		cmdExecutor = createMockExecutor(sampleCPUData, sampleOSData)
		parsedItems := collectPlatformDependentInstanceData()
		assert.Equal(t, len(parsedItems), 1)
		assert.Equal(t, sampleDataWindowsParsed[i], parsedItems[0])
	}
}

func TestCollectPlatformDependentInstanceDataWithError(t *testing.T) {
	cmdExecutor = MockTestExecutorWithError
	parsedItems := collectPlatformDependentInstanceData()
	assert.Equal(t, len(parsedItems), 0)
}
