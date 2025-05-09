package signature

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/tjfoc/gmsm/sm2"
	gmsm_x509 "github.com/tjfoc/gmsm/x509"
)

func Test_parsePublicKey(t *testing.T) {
	dataNeedVerify := "this is a test line."
	for _, algo := range []string{algorithm_sm3withsm2, algorithm_sha256withrsa} {
		var signature, publicKeyStr string
		var err error
		switch algo {
		case algorithm_sm3withsm2:
			signature, publicKeyStr, err = signWithSm2(dataNeedVerify)
		case algorithm_sha256withrsa:
			signature, publicKeyStr, err = signWithRsa(dataNeedVerify)
		default:
			assert.Fail(t, "unknown algorthm")
		}
		assert.Nil(t, err)

		cert := &Cert{
			KeypairVersion:   1,
			SignatureVersion: 2,
			PublicKeyStr:     publicKeyStr,
			Algorithm:        algo,
		}
		err = cert.parsePublicKey()
		assert.Nil(t, err)
		signatureByte, err := base64.StdEncoding.DecodeString(signature)

		var res bool
		switch algo {
		case algorithm_sm3withsm2:
			res, err = verifyWithSm3withsm2(cert.PublicKeySm2, dataNeedVerify, signatureByte)
		case algorithm_sha256withrsa:
			res, err = verifyWithSha256withrsa(cert.PublicKeyRsa, dataNeedVerify, signatureByte)
		default:
			assert.Fail(t, "unknown algorthm")
		}
		assert.Nil(t, err)
		assert.True(t, res)
	}

}

func signWithSm2(data string) (signature, publicKeyStr string, err error) {
	var privateKey *sm2.PrivateKey
	privateKey, err = sm2.GenerateKey(nil)
	if err != nil {
		return
	}

	// public key
	var X509PublicKey []byte
	X509PublicKey, err = gmsm_x509.MarshalSm2PublicKey(&privateKey.PublicKey)
	if err != nil {
		return
	}
	pblicBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: X509PublicKey,
	}
	buf := bytes.NewBufferString("")
	if err = pem.Encode(buf, &pblicBlock); err != nil {
		return
	}
	publicKeyStr = buf.String()

	// signature
	var sign []byte
	sign, err = privateKey.Sign(nil, []byte(data), nil)
	if err != nil {
		return
	}
	signature = base64.StdEncoding.EncodeToString(sign)

	return
}

func signWithRsa(data string) (signature, publicKeyStr string, err error) {
	var privateKey *rsa.PrivateKey
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	// public key
	publicKey := privateKey.PublicKey
	var X509PublicKey []byte
	X509PublicKey, err = x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return
	}
	pblicBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: X509PublicKey,
	}
	buf := bytes.NewBufferString("")
	if err = pem.Encode(buf, &pblicBlock); err != nil {
		return
	}
	publicKeyStr = buf.String()

	// signature
	msgHash := crypto.SHA256.New()
	if _, err = msgHash.Write([]byte(data)); err != nil {
		return
	}
	msgHashSum := msgHash.Sum(nil)
	var sign []byte
	sign, err = rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, msgHashSum)
	if err != nil {
		return
	}
	signature = base64.StdEncoding.EncodeToString(sign)

	return
}

func TestSig(t *testing.T) {
	return
	guard := gomonkey.ApplyFunc(pathutil.GetCrossVersionConfigPath, func() (string, error) {
		curr, _ := os.Executable()
		curr = filepath.Dir(curr)
		return curr, nil
	})
	defer guard.Reset()

	guard_1 := gomonkey.ApplyFunc(util.GetInstanceId, func() string {
		return "i-abc"
	})
	defer guard_1.Reset()

	taskInfo := &models.RunTaskInfo{
		TaskId:    "t-xxx",
		Content:   "cHMgLWVm", // ps -ef
		Signature: "2#1#",
		UserId:    "1234556",
	}
	ok, err := VerifyTaskSign(logrus.New(), *taskInfo)
	fmt.Println(ok, err)
}
