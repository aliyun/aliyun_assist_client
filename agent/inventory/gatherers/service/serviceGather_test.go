package service

import (
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testService = []model.ServiceData{
	{
		Name:               "BrokerInfrastructure",
		DisplayName:        "Background Tasks Infrastructure Service",
		Status:             "Running",
		DependentServices:  "embeddedmode",
		ServicesDependedOn: "DcomLaunch RpcSs RpcEptMapper",
		ServiceType:        "Win32ShareProcess",
		StartType:          "",
	},
	{
		Name:               "embeddedmode",
		DisplayName:        "Embedded Mode",
		Status:             "Stopped",
		DependentServices:  "",
		ServicesDependedOn: "BrokerInfrastructure",
		ServiceType:        "Win32ShareProcess",
		StartType:          "",
	},
}

func testCollectServiceData(config model.Config) (data []model.ServiceData, err error) {
	return testService, nil
}

func TestServiceGather(t *testing.T) {
	_, err := testCollectServiceData(model.Config{})
	model := model.Config{}
	g := Gatherer()
	item, _ := g.Run(model)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(item))
	assert.Equal(t, GathererName, item[0].Name)
	assert.Equal(t, SchemaVersionOfServiceGatherer, item[0].SchemaVersion)
}
