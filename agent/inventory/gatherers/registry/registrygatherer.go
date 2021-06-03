package registry

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	// GathererName captures name of Registry gatherer
	GathererName = "ACS:WindowsRegistry"
	// SchemaVersionOfRegistryGatherer represents schema version of Registry gatherer
	SchemaVersionOfRegistryGatherer = "1.0"
)

type T struct{}

// Gatherer returns new Process gatherer
func Gatherer() *T {
	return new(T)
}

var collectData = collectRegistryData

// Name returns name of Process gatherer
func (t *T) Name() string {
	return GathererName
}

// Run executes Registry gatherer and returns list of inventory.Item comprising of registry data
func (t *T) Run(configuration model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run windows registry gatherer begin")
	var result model.Item

	//CaptureTime must comply with format: 2016-07-30T18:15:37Z
	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)
	var data []model.RegistryData
	data, err = collectData(configuration)
	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersionOfRegistryGatherer,
		Content:       data,
		CaptureTime:   captureTime,
	}

	items = append(items, result)
	log.GetLogger().Infof("run windows registry gatherer end, got %d registry", len(data))
	return
}
