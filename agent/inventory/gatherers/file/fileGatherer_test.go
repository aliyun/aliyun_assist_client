package file

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

func TestGatherer(t *testing.T) {

	var items []model.Item

	config := model.Config{
		Collection: "Enabled",
		Filters:    `[{"Path": "$HOME","Pattern":["*.txt"],"Recursive": false}]`,
		Location:   "",
	}
	g := Gatherer()
	items, err := g.Run(config)
	assert.Nil(t, err, "Unexpected error thrown")
	assert.Equal(t, 1, len(items), "File Gather should return 1 inventory type data.")
}
