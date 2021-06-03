// +build windows

package gatherers

import (
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/application"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/file"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/network"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/registry"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/role"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/service"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/windowsupdate"
)

var supportedGathererNames = []string{
	application.GathererName,
	network.GathererName,
	file.GathererName,
	service.GathererName,
	windowsupdate.GathererName,
	role.GathererName,
	registry.GathererName,
}


