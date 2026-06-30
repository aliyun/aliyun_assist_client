package encryptionutil

import (
	"encoding/base64"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/kms"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/paramstore"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestKMSDecrypt(t *testing.T) {
	dataKey := "1234567890123456"
	dataKeyBase64 := base64.StdEncoding.EncodeToString([]byte(dataKey))

	defer gomonkey.ApplyFunc(paramstore.GetRoleName, func() (string, error) { return "testRoleName", nil }).Reset()
	defer gomonkey.ApplyFunc(util.GetRegionId, func() string { return "cn-hangzhou" }).Reset()
	defer gomonkey.ApplyFunc(kms.NewClientWithEcsRamRole, func(_ string, _ string) (*kms.Client, error) { return &kms.Client{}, nil }).Reset()
	logger := log.GetLogger().WithFields(logrus.Fields{
		"sessionType": "shell",
		"sessionId":   "testId",
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Print", func(_ logrus.FieldLogger, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Errorf", func(_ logrus.FieldLogger, _ string, _ ...interface{}) {}).Reset()
	var client *kms.Client
	defer gomonkey.ApplyMethod(reflect.TypeOf(client), "Decrypt", func(_ *kms.Client, _ *kms.DecryptRequest) (*kms.DecryptResponse, error) {
		return &kms.DecryptResponse{Plaintext: dataKeyBase64}, nil
	}).Reset()

	gotDataKey, gotErr := KMSDecrypt(logger, dataKey)
	assert.Equal(t, dataKey, gotDataKey)
	assert.Nil(t, gotErr)
}
