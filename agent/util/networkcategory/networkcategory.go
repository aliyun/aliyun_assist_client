package networkcategory

import (
	"sync/atomic"
)

type NetworkCategory string
const (
	NetworkVPC NetworkCategory = "vpc"
	NetworkClassic NetworkCategory = "classic"
	NetworkHybrid NetworkCategory = "hybrid"
	// In some cloud environment the metaserver(100.100.100.200) is provided,
	// and axt server domain can be retrieved from http://100.100.100.200/latest/global-config/aliyun-assist-server-url ,
	// which is slightly different with traditional resolution process in VPC
	// network. Thus a separate category indicator is here.
	// See getDomainbyMetaServer() function for more details.
	NetworkWithMetaserver NetworkCategory = "with-metaserver"

	NetworkCategoryUnknown NetworkCategory = "unknown"
)

var (
	// _networkCategory would be determined when detecting region id (via
	// initRegionId function in hostfinder.go) or retriving domain from
	// metaserver (via getDomainbyMetaServer function in hostfinder.go), thus
	// atomic.Value with NetworkCategory content is used for concurrency control.
	_neverDirectRW_atomic_networkCategory atomic.Value
)

// Set wraps atomic store action for network category indicator
func Set(category NetworkCategory) {
	_neverDirectRW_atomic_networkCategory.Store(category)
}

// Get returns detected category of network environment during
// detecting region id for determined axt server domain, otherwise NetworkCategoryUnknown
func Get() NetworkCategory {
	networkCategory, ok := _neverDirectRW_atomic_networkCategory.Load().(NetworkCategory)
	if !ok {
		return NetworkCategoryUnknown
	}

	if networkCategory == "" {
		return NetworkCategoryUnknown
	}
	return networkCategory
}
