package resources
import (
	"testing"
	"encoding/json"

	"github.com/stretchr/testify/assert"
)


func TestLoad(t *testing.T) {
	propStr := `
	{
		"Policy": {
			"ACS:InstanceInformation": {
				"Collection": "true"
			},
			"ACS:File": {
				"Collection": "true",
				"Filters": "[{\"Path\": \"/home/admin/test\",\"Pattern\":[\"*\"],\"Recursive\":false}]"
			}
		}
	}
	`
	var properties map[string]interface{} = make(map[string]interface{})
	json.Unmarshal([]byte(propStr), &properties)

	inventory := &InventoryState{}
	err := inventory.Load(properties)
	assert.Nil(t, err)

	assert.Equal(t, inventory.InventoryPolicy["ACS:InstanceInformation"].Collection, "true")
	assert.Equal(t, inventory.InventoryPolicy["ACS:InstanceInformation"].Filters, "")
	assert.Equal(t, inventory.InventoryPolicy["ACS:InstanceInformation"].Location, "")
	assert.Equal(t, inventory.InventoryPolicy["ACS:File"].Collection, "true")
	assert.Equal(t, inventory.InventoryPolicy["ACS:File"].Filters, "[{\"Path\": \"/home/admin/test\",\"Pattern\":[\"*\"],\"Recursive\":false}]")
	assert.Equal(t, inventory.InventoryPolicy["ACS:File"].Location, "")
}