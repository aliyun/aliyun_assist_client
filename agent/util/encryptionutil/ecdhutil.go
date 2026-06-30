package encryptionutil

import (
	"crypto/ecdh"
	"crypto/rand"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

var curv ecdh.Curve = ecdh.P256()

func GeneratePublicKey(logger logrus.FieldLogger) (*ecdh.PrivateKey, []byte, error) {
	privateKey, err := curv.GenerateKey(rand.Reader)
	if err != nil {
		return &ecdh.PrivateKey{}, []byte{}, err
	}
	logger.Infof("Agent ECDH private key: %v", privateKey.Bytes())
	publicKey := privateKey.PublicKey().Bytes()
	return privateKey, publicKey, nil
}

func GenerateSharedSecret(agentPrivateKey *ecdh.PrivateKey, clientPublicKeyByte []byte) ([]byte, error) {
	clientPublicKey, err := curv.NewPublicKey(clientPublicKeyByte)
	if err != nil {
		return []byte{}, err
	}
	sharedSecret, err := agentPrivateKey.ECDH(clientPublicKey)
	if err != nil {
		return []byte{}, err
	}
	return sharedSecret, nil
}
