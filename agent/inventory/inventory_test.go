package inventory

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/stretchr/testify/assert"
)

func TestRunGatherers(t *testing.T) {

	var err error
	var items []model.Item

	policy := &model.Policy{}
	policy.InventoryPolicy = map[string]model.Config{
		/*"ACS:InstanceInformation": model.Config{
			Collection: "Enabled",
			Filters:    "",
			Location:   "",
		},
		"ACS:Network": model.Config{
			Collection: "Enabled",
			Filters:    "",
			Location:   "",
		},*/
		"ACS:File": model.Config{
			Collection: "Disable",
			Filters:    `[{"Path": "$HOME","Pattern":["*.txt"],"Recursive": false}]`,
			Location:   "",
		},
	}

	items, err = RunGatherers(*policy)

	assert.Nil(t, err, "Unexpected error thrown")
	assert.Equal(t, 0, len(items), "Custom Gather should return 2 inventory type data.")

}
