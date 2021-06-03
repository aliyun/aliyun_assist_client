// +build darwin freebsd linux netbsd openbsd

package gatherers

import (
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/application"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/file"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/network"
)

var supportedGathererNames = []string{
	application.GathererName,
	network.GathererName,
	file.GathererName,
}
