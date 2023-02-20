package service

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"

	"github.com/stretchr/testify/assert"
)

var testServiceOutput = "[{\"Name\": \"Router\", \"DisplayName\": \"Router Service\", \"Status\": \"Stopped\", \"DependentServices\": \"\", \"ServicesDependedOn\": \"\", \"ServiceType\": \"Win32ShareProcess\", \"StartType\": \"\"},{\"Name\": \"ALG\", \"DisplayName\": \"Application Layer Gateway Service\", \"Status\": \"Stopped\", \"DependentServices\": \"\", \"ServicesDependedOn\": \"BrokerInfrastructure\", \"ServiceType\": \"Win32OwnProcess\", \"StartType\": \"\"}]"
var testServiceOutputIncorrect = "[{\"Name\": \"<start123>AJRouter\", \"DisplayName\": \"Router Service\", \"Status\": \"Stopped\", \"DependentServices\": \"\", \"ServicesDependedOn\": \"\", \"ServiceType\": \"Win32ShareProcess\", \"StartType\": \"\"},{\"Name\": \"ALG\", \"DisplayName\": \"Application Layer Gateway Service\", \"Status\": \"Stopped\", \"DependentServices\": \"\", \"ServicesDependedOn\": \"BrokerInfrastructure\", \"ServiceType\": \"Win32OwnProcess\", \"StartType\": \"\"}]"

var testServiceOutputData = []model.ServiceData{
	{
		Name:               "Router",
		DisplayName:        "Router Service",
		Status:             "Stopped",
		DependentServices:  "",
		ServicesDependedOn: "",
		ServiceType:        "Win32ShareProcess",
		StartType:          "",
	},
	{
		Name:               "ALG",
		DisplayName:        "Application Layer Gateway Service",
		Status:             "Stopped",
		DependentServices:  "",
		ServicesDependedOn: "BrokerInfrastructure",
		ServiceType:        "Win32OwnProcess",
		StartType:          "",
	},
}

var testServiceOutputDataEmpty = []model.ServiceData{}

// type Mock struct {
// 	mock.Mock
// }

// func NewMockDefault() *Mock

func createMockTestExecuteCommand(output string, err error) func(string, ...string) ([]byte, error) {

	return func(string, ...string) ([]byte, error) {
		return []byte(output), err
	}
}

func TestRandomString(t *testing.T) {
	resp := randomString(5)
	assert.NotNil(t, resp)
}

func TestMatk(t *testing.T) {
	resp := mark("test")
	assert.NotNil(t, resp)
}

func TestCollectServiceData(t *testing.T) {
	// ctx := NewMockDefault()
	cmdExecutor = createMockTestExecuteCommand(testServiceOutput, nil)
	data, err := collectServiceData(model.Config{})
	print(data)
	if err != nil {
		print("error : %v", err.Error())
	}
	assert.Equal(t, testServiceOutputData, data)
	// assert.NotNil(t, err)
}

func TestServiceDataCmdErr(t *testing.T) {
	cmdExecutor = createMockTestExecuteCommand("", errors.New("error"))

	data, err := collectServiceData(model.Config{})

	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Command failed with error")
	assert.Equal(t, testServiceOutputDataEmpty, data)
}
func TestServiceDataInvalidOutput(t *testing.T) {

	cmdExecutor = createMockTestExecuteCommand("Invalid", nil)
	data, err := collectServiceData(model.Config{})

	assert.NotNil(t, err)
	assert.Equal(t, testServiceOutputDataEmpty, data)
}

func TestServiceDataInvalidMarker(t *testing.T) {
	startMarker = "<start123>"
	endMarker = "<test>"
	cmdExecutor = createMockTestExecuteCommand(testServiceOutputIncorrect, nil)

	data, err := collectServiceData(model.Config{})

	assert.NotNil(t, err)
	assert.Equal(t, testServiceOutputDataEmpty, data)
}
