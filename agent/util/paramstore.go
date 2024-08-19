package util

import (
	"fmt"
	"time"
	"errors"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/oos"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
)

var (
	cacheRamRole = ""
	cacheRamRoleErr error
	lastUpdateTime time.Time
)

var (
	ErrRoleNameFailed = errors.New("RoleNameFailed")
	ErrParameterStoreNotAccessible = errors.New("ParameterStoreNotAccessible")
	ErrParameterFailed = errors.New("ParameterFailed")
)


func GetRoleName() (string, error) {
	url := "http://100.100.100.200/latest/meta-data/ram/security-credentials/"
	cacheRamRoleErr, cacheRamRole = HttpGet(url)
	if cacheRamRoleErr != nil {
		cacheRamRole = ""
	}
	lastUpdateTime = time.Now()
	return cacheRamRole, cacheRamRoleErr
}

func GetRoleNameTtl(ttl time.Duration) (string, error) {
	if time.Since(lastUpdateTime) >= ttl {
		return GetRoleName()
	}
	return cacheRamRole, cacheRamRoleErr
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
