package role

import (
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

const (
	// GathererName captures name of Role gatherer
	GathererName = "ACS:WindowsRole"
	// SchemaVersionOfRoleGatherer represents schema version of Role gatherer
	SchemaVersionOfRoleGatherer = "1.0"
	RoleCountLimit              = 500
	RoleCountLimitExceeded      = "Role Count Limit Exceeded"
)

type T struct{}

// Gatherer returns new Role gatherer
func Gatherer() *T {
	return new(T)
}

var collectData = collectRoleData

// Name returns name of Process gatherer
func (t *T) Name() string {
	return GathererName
}

// Run executes Role gatherer and returns list of inventory.Item comprising of role data
func (t *T) Run(configuration model.Config) (items []model.Item, err error) {
	log.GetLogger().Info("run role gatherer begin")
	var result model.Item

	currentTime := time.Now().UTC()
	captureTime := currentTime.Format(time.RFC3339)
	var data []model.RoleData
	data, err = collectData(configuration)
	if err != nil {
		log.GetLogger().WithError(err).Error("Role gather err")
	}
	result = model.Item{
		Name:          t.Name(),
		SchemaVersion: SchemaVersionOfRoleGatherer,
		Content:       data,
		CaptureTime:   captureTime,
	}
	items = append(items, result)
	log.GetLogger().Infof("run role gatherer end, got %d roles", len(data))
	return
}
