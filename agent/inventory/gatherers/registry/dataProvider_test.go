package registry

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testRegistryOutput = "[{}]"

var testRegistryOutputData = []model.RegistryData(nil)

func createMockExecutor(output []string, err []error) func(string, ...string) ([]byte, error) {
	var index = 0
	return func(string, ...string) ([]byte, error) {
		if index < len(output) {
			index += 1
		}
		return []byte(output[index-1]), err[index-1]
	}
}

func TestGetRegistryData(t *testing.T) {

	cmdExecutor = createMockExecutor([]string{testRegistryOutput}, []error{nil})
	startMarker = "<start1234>"
	endMarker = "<end1234>"
	mockFilters := "[]"
	mockConfig := model.Config{Collection: "Enabled", Filters: mockFilters, Location: ""}
	data, err := collectRegistryData(mockConfig)

	assert.Nil(t, err)
	assert.Equal(t, testRegistryOutputData, data)
}
