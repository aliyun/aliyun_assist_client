package resources

import (
	"encoding/json"

	"github.com/aliyun/aliyun_assist_client/agent/inventory"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

type InventoryState model.Policy

func (is *InventoryState) Load(properties map[string]interface{}) (err error) {
	data, err := json.Marshal(properties)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, is)
	return
}

func (is *InventoryState) Apply() (status string, extraInfo string, err error) {
	return is.Monitor()
}

func (is *InventoryState) Monitor() (status string, extraInfo string, err error) {
	_, err = inventory.RunGatherers(model.Policy(*is))
	if err != nil {
		return Failed, "", err
	}
	return Compliant, "", nil
}
