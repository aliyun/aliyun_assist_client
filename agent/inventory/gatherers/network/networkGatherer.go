package network

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

const (
	// GathererName captures name of application gatherer
	GathererName = "ACS:Network"
	// SchemaVersionOfApplication represents schema version of application gatherer
	SchemaVersionOfNetwork          = "1.0"
	NetworkConfigCountLimit         = 500
	NetworkConfigCountLimitExceeded = "Network Configuration Count Limit Exceeded"
)

type T struct{}

func Gatherer() *T {
	return new(T)
}

func (t *T) Name() string {
	return GathererName
}

func (t *T) Run(config model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run network gatherer begin")
	var result model.Item

	//CaptureTime must comply with format: 2016-07-30T18:15:37Z to comply with regex at OOS.
	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)

	data := collectNetworkData(config)
	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersionOfNetwork,
		Content:       data,
		CaptureTime:   captureTime,
	}

	items = append(items, result)
	log.GetLogger().Infof("run network gatherer end, got %d network configurations", len(data))
	return
}
