package util

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/oos"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/networkcategory"
)

func GetRoleName() (string, error) {
	url := "http://100.100.100.200/latest/meta-data/ram/security-credentials/"
	err, roleName := HttpGet(url)
	if err != nil {
		roleName = ""
	}
	return roleName, err
}

func GetSecretParam(name string) (string, error) {
	region := GetRegionId()
	roleName, err := GetRoleName()
	if err != nil {
		log.GetLogger().Errorln("GetRoleName failed ", "error:", err.Error())
		errMsg := fmt.Sprintf("Get role name failed: %s.", err.Error())
		return errMsg, ErrRoleNameFailed
	}

	ecs_client, err := oos.NewClientWithEcsRamRole(region, roleName)
	if err != nil {
		log.GetLogger().Errorln("NewClientWithEcsRamRole failed:", roleName, " error:", err.Error())
		errMsg := fmt.Sprintf("Create new client with ecs ram role %s failed: %s.", roleName, err.Error())
		return errMsg, ErrParameterStoreNotAccessible
	}
	if networkcategory.Get() == networkcategory.NetworkVPC {
		ecs_client.Network = "vpc"
	}

	// GetSecretParameter
	request := oos.CreateGetSecretParameterRequest()
	request.Name = name
	request.WithDecryption = "true"

	response, err := ecs_client.GetSecretParameter(request)
	if err != nil {
		log.GetLogger().Errorln("GetSecretParameter failed:", roleName, " error:", err.Error())
		errMsg := fmt.Sprintf("Get secret parameter '%s' with ecs ram role %s failed: %s.", name, roleName, err.Error())
		return errMsg, ErrParameterFailed
	}

	value := response.Parameter.Value

	return value, err
}

func GetParam(name string) (string, error) {
	region := GetRegionId()
	roleName, err := GetRoleName()
	if err != nil {
		log.GetLogger().Errorln("GetRoleName failed ", "error:", err.Error())
		errMsg := fmt.Sprintf("Get role name failed: %s.", err.Error())
		return errMsg, ErrRoleNameFailed
	}

	ecs_client, err := oos.NewClientWithEcsRamRole(region, roleName)
	if err != nil {
		log.GetLogger().Errorln("NewClientWithEcsRamRole failed:", roleName, " error:", err.Error())
		errMsg := fmt.Sprintf("Create new client with ecs ram role %s failed: %s.", roleName, err.Error())
		return errMsg, ErrParameterStoreNotAccessible
	}
	if networkcategory.Get() == networkcategory.NetworkVPC {
		ecs_client.Network = "vpc"
	}

	request := oos.CreateGetParameterRequest()
	request.Name = name
	request.Scheme = "https"

	response, err := ecs_client.GetParameter(request)
	if err != nil {
		log.GetLogger().Errorln("GetParameter failed:", roleName, " error:", err.Error())
		errMsg := fmt.Sprintf("Get parameter '%s' with ecs ram role %s failed: %s.", name, roleName, err.Error())
		return errMsg, ErrParameterFailed
	}

	value := response.Parameter.Value

	return value, err
}
