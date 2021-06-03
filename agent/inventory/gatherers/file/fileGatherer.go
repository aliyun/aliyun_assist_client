package file

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	// GathererName captures name of file gatherer
	GathererName = "ACS:File"
	// SchemaVersionOfFileGatherer represents schema version of file gatherer
	SchemaVersionOfFileGatherer = "1.0"
)

type T struct{}

// Gatherer returns new file gatherer
func Gatherer() *T {
	return new(T)
}

// Name returns name of file gatherer
func (t *T) Name() string {
	return GathererName
}

// Run executes file gatherer and returns list of inventory.Item comprising of file data
func (t *T) Run(config model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run file gatherer begin")
	var result model.Item

	//CaptureTime must comply with format: 2016-07-30T18:15:37Z to comply with regex at OOS.
	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)
	var data []model.FileData
	data, err = collectFileData(config)

	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersionOfFileGatherer,
		Content:       data,
		CaptureTime:   captureTime,
	}

	items = append(items, result)
	log.GetLogger().Infof("run file gatherer end, got %d files", len(data))
	return
}
