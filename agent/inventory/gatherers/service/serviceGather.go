package service

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	// GathererName captures name of Service gatherer
	GathererName = "ACS:Service"
	// SchemaVersionOfServiceGatherer represents schema version of Service gatherer
	SchemaVersionOfServiceGatherer = "1.0"
	ServiceCountLimit              = 500
	ServiceCountLimitExceeded      = "Service Count Limit Exceeded"
)

type T struct{}

// Gatherer returns new Process gatherer
func Gatherer() *T {
	return new(T)
}

var collectData = collectServiceData

// Name returns name of Process gatherer
func (t *T) Name() string {
	return GathererName
}

// Run executes Service gatherer and returns list of inventory.Item comprising of service data
func (t *T) Run(configuration model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run service gatherer begin")
	var result model.Item

	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)
	var data []model.ServiceData
	data, err = collectData(configuration)

	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersionOfServiceGatherer,
		Content:       data,
		CaptureTime:   captureTime,
	}
	items = append(items, result)
	log.GetLogger().Infof("run service gatherer end, got %d services", len(data))
	return
}
