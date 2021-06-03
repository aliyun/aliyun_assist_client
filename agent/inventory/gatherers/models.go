package gatherers

import (
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/application"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/file"
	instancedetailedinfo "github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/instancedetailedinformation"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/network"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/registry"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/role"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/service"
	windowsupdate "github.com/aliyun/aliyun_assist_client/agent/inventory/gatherers/windowsupdate"
	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

type T interface {
	//returns the Name of the gatherer
	Name() string
	Run(config model.Config) (items []model.Item, err error)
}

type SupportedGatherer map[string]T
type InstalledGatherer map[string]T

func InitializeGatherers() (SupportedGatherer, InstalledGatherer) {
	var installedGathererNames []string
	installedGatherer := InstalledGatherer{
		application.GathererName:          application.Gatherer(),
		network.GathererName:              network.Gatherer(),
		file.GathererName:                 file.Gatherer(),
		service.GathererName:              service.Gatherer(),
		windowsupdate.GathererName:        windowsupdate.Garherer(),
		registry.GathererName:             registry.Gatherer(),
		role.GathererName:                 role.Gatherer(),
		instancedetailedinfo.GathererName: instancedetailedinfo.Gatherer(),
	}

	for key := range installedGatherer {
		installedGathererNames = append(installedGathererNames, key)
	}

	supportedGatherer := SupportedGatherer{}

	for _, name := range supportedGathererNames {
		supportedGatherer[name] = installedGatherer[name]
	}

	return supportedGatherer, installedGatherer
}
