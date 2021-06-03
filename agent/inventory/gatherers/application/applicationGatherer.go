package application

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/util/jsonutil"
)

const (
	// GathererName captures name of application gatherer
	GathererName = "ACS:Application"
	// SchemaVersionOfApplication represents schema version of application gatherer
	SchemaVersionOfApplication = "1.0"
)

type T struct{}

func Gatherer() *T {
	return new(T)
}

func (t *T) Name() string {
	return GathererName
}

func (t *T) Run(config model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run application gatherer begin")

	var result model.Item

	//CaptureTime must comply with format: 2016-07-30T18:15:37Z to comply with regex at OOS.
	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)

	data := collectApplicationData(config)
	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersionOfApplication,
		Content:       data,
		CaptureTime:   captureTime,
	}
	output, _ := jsonutil.Marshal(result)
	log.GetLogger().Debugf("output application gather: %s", output)
	items = append(items, result)
	log.GetLogger().Infof("run application gatherer end, got %d applications", len(data))
	return
}
