package encryptionutil

import (
	"crypto/ecdh"
	"crypto/rand"
	"reflect"
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestGeneratePiblicKey(t *testing.T) {
	logger := log.GetLogger().WithFields(logrus.Fields{
		"sessionType": "shell",
		"sessionId":   "testId",
	})
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Infof", func(_ logrus.FieldLogger, _ string, _ ...interface{}) {}).Reset()

	privateKey, publicKey, err := GeneratePublicKey(logger)

	assert.NotNil(t, privateKey)
	assert.NotNil(t, publicKey)
	assert.Nil(t, err)
}

func TestGenerateSharedSecret(t *testing.T) {
	curv := ecdh.P256()
	clientPrivateKey, _ := curv.GenerateKey(rand.Reader)
	clientPublicKey := clientPrivateKey.PublicKey().Bytes()
	logger := log.GetLogger().WithFields(logrus.Fields{
		"sessionType": "shell",
		"sessionId":   "testId",
	})
	agentPrivateKey, _, _ := GeneratePublicKey(logger)

	sharedSecret, err := GenerateSharedSecret(agentPrivateKey, clientPublicKey)

	assert.NotNil(t, sharedSecret)
	assert.Nil(t, err)
}
