package instancedetailedinformation

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/stretchr/testify/assert"
)

func DataGenerator() []model.InstanceDetailedInformation {
	return []model.InstanceDetailedInformation{
		{
			CPUModel:              "Intel(R) Xeon(R) CPU E5-2686 v4 @ 2.30GHz",
			CPUSpeedMHz:           "1772",
			CPUs:                  "64",
			CPUSockets:            "2",
			CPUCores:              "32",
			CPUHyperThreadEnabled: "true",
		},
	}
}

func TestGatherer(t *testing.T) {
	g := Gatherer()
	collectData = DataGenerator
	items, err := g.Run(model.Config{})
	assert.Nil(t, err, "Unexpected error thrown")
	assert.Equal(t, 1, len(items))
	assert.Equal(t, items[0].Name, g.Name())
	assert.Equal(t, items[0].SchemaVersion, SchemaVersion)
	assert.Equal(t, items[0].Content, DataGenerator())
	assert.NotNil(t, items[0].CaptureTime)
}
