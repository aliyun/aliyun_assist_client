package encryptionutil

import (
	"encoding/base64"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/paramstore"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

func KMSDecrypt(logger logrus.FieldLogger, cipherText string) (string, error) {
	roleName, err := paramstore.GetRoleName()
	if err != nil {
		logger.Errorf("Get role name failed: %s", err)
		return "", err
	}
	logger.Print("role name: ", roleName)

	regionId := util.GetRegionId()
	ecs_client, err := kms.NewClientWithEcsRamRole(regionId, roleName)
	if err != nil {
		logger.Errorf("Get Region Id failed: %s", err)
		return "", err
	}
	ecs_client.Network = "vpc"

	request := kms.CreateDecryptRequest()
	request.SetScheme("HTTPS")
	request.CiphertextBlob = cipherText

	response, err := ecs_client.Decrypt(request)
	if err != nil {
		logger.Errorf("ESC_Client decryption failed: %s", err)
		return "", err
	}
	dataKey := response.Plaintext

	var dataKeyByte []byte
	if dataKeyByte, err = base64.StdEncoding.DecodeString(dataKey); err != nil {
		logger.Errorf("ESC_Client decryption failed: %s", err)
		return "", err
	}

	return string(dataKeyByte), nil
}
