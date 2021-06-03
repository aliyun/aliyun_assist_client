package instancedetailedinformation

import (
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

// CollectInstanceData collects data from the system using platform specific queries.
func CollectInstanceData() []model.InstanceDetailedInformation {
	return collectPlatformDependentInstanceData()
}
