package checknet

import (
	"sync/atomic"

	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
)

type _NetworkCategoryCache struct {
	neverDirectRW_atomic_networkCategory atomic.Value
}

var (
	networkCategoryCache _NetworkCategoryCache
)

// Set wraps atomic store action for network category indicator
func (c *_NetworkCategoryCache) Set(category networkcategory.NetworkCategory) {
	c.neverDirectRW_atomic_networkCategory.Store(category)
}

// Get returns detected category of network environment during
// detecting region id for determined axt server domain, otherwise NetworkCategoryUnknown
func (c *_NetworkCategoryCache) Get() networkcategory.NetworkCategory {
	networkCategory, ok := c.neverDirectRW_atomic_networkCategory.Load().(networkcategory.NetworkCategory)
	// Just use cache if it is valid
	if ok && networkCategory != "" {
		return networkCategory
	}

	// Otherwise, return value in networkcategory module
	return networkcategory.Get()
}
