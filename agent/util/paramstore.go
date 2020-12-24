package util

import (
	"errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/oos"
)

func getRoleName() (string, error) {
	url := "http://100.100.100.200/latest/meta-data/ram/security-credentials/"
	err, roleName := HttpGet(url)
	return roleName,err
}

func GetSecretParam(name string) (string, error) {
	err, region := getRegionIdInVpc()
	if err != nil {
		return "", errors.New("RegionIdFailed")
	}

	roleName, err := getRoleName()
	if err != nil {
		return "", errors.New("RoleNameFailed")
	}

	ecs_client, err := oos.NewClientWithEcsRamRole(region, roleName)
	if err != nil {
		return "", errors.New("RoleNameFailed")
	}

	// GetSecretParameter
	request := oos.CreateGetSecretParameterRequest()
	request.Name = name
	request.WithDecryption = "true"

	response, err := ecs_client.GetSecretParameter(request)
	if err != nil {
		return "", errors.New("ParameterFailed")
	}

	value := response.Parameter.Value

    return value,err
}