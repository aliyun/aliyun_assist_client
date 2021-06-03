// Package instancedetailedinformation contains a gatherer for the ACS:InstanceDetailedInformation inventory type.
package instancedetailedinformation

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	// GathererName captures name of gatherer
	GathererName = "ACS:InstanceDetailedInformation"
	// SchemaVersion represents the schema version of this gatherer
	SchemaVersion = "1.0"
)

// T represents the gatherer type, which implements all contracts for gatherers.
type T struct{}

// decoupling for easy testability
var collectData = CollectInstanceData

// Gatherer returns new application gatherer
func Gatherer() *T {
	return new(T)
}

// Name returns name of application gatherer
func (t *T) Name() string {
	return GathererName
}

// Run executes the gatherer and returns list of inventory.Item comprising of collected data
func (t *T) Run(configuration model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run InstanceDetailedInformation gatherer begin")
	var result model.Item

	//CaptureTime must comply with format: 2016-07-30T18:15:37Z.
	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)

	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersion,
		Content:       collectData(),
		CaptureTime:   captureTime,
	}

	items = append(items, result)
	log.GetLogger().Info("run InstanceDetailedInformation gatherer end")
	return
}
