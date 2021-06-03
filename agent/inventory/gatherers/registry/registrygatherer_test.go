package registry

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testRegistry = []model.RegistryData{
	{
		ValueName: "abc",
		ValueType: "REG_SZ",
		KeyPath:   "HKEY_LOCAL_MACHINE\\SOFTWARE",
		Value:     "pqr",
	},
	{
		ValueName: "ad",
		ValueType: "REG_DWORD",
		KeyPath:   "HKEY_LOCAL_MACHINE\\SOFTWARE",
		Value:     "1000",
	},
}

func testCollectRegistryData(config model.Config) (data []model.RegistryData, err error) {
	return testRegistry, nil
}

func TestGatherer(t *testing.T) {
	gatherer := Gatherer()
	collectData = testCollectRegistryData
	item, err := gatherer.Run(model.Config{})
	assert.Nil(t, err)
	assert.Equal(t, 1, len(item))
	assert.Equal(t, GathererName, item[0].Name)
	assert.Equal(t, SchemaVersionOfRegistryGatherer, item[0].SchemaVersion)
	assert.Equal(t, testRegistry, item[0].Content)
}
