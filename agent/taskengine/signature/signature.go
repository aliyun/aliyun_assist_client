package signature

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	gmsm_x509 "github.com/tjfoc/gmsm/x509"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/tjfoc/gmsm/sm2"
)

const (
	certFileName = "task_sign_certs"

	algorithm_sm3withsm2    = "SM3WithSM2"
	algorithm_sha256withrsa = "SHA256WithRSA"
)

type Cert struct {
	KeypairVersion   int            `json:"keypairVersion"`
	SignatureVersion int            `json:"signatureVersion"`
	PublicKeyStr     string         `json:"publicKey"`
	Algorithm        string         `json:"algorithm"`
	SignatureFields  []string       `json:"signatureFields"`
	PublicKeyRsa     *rsa.PublicKey `json:"-"`
	PublicKeySm2     *sm2.PublicKey `json:"-"`
}

type SignatureCertificateReq struct {
	KeypairVersion   int `json:"keypairVersion"`
	SignatureVersion int `json:"signatureVersion"`
}

var (
	ErrorUnknownSignatureFormat = errors.New("unknown signature format")
	ErrorCertNotFound           = errors.New("cert not found")
	ErrorUnknownSignAlgorithm   = errors.New("unknown sign algorithm")
	ErrorParsePublicKey         = errors.New("parse public key failed")
	ErrorUnknownInstanceId      = errors.New("unknown instanceId")

	certs_ sync.Map

	certsMp     map[string]*Cert
	certsMpLock sync.Mutex

	loadCertsOnce sync.Once
)

func (c *Cert) parsePublicKey() error {
	switch c.Algorithm {
	case algorithm_sha256withrsa:
		block, _ := pem.Decode([]byte(c.PublicKeyStr))
		if block == nil {
			return ErrorParsePublicKey
		}
		pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return err
		}
		c.PublicKeyRsa = pubInterface.(*rsa.PublicKey)
	case algorithm_sm3withsm2:
		block, _ := pem.Decode([]byte(c.PublicKeyStr))
		if block == nil {
			return ErrorParsePublicKey
		}
		var err error
		c.PublicKeySm2, err = gmsm_x509.ParseSm2PublicKey(block.Bytes)
		if err != nil {
			return err
		}
	default:
		return ErrorUnknownSignAlgorithm
	}
	return nil
}

func VerifyTaskSign(logger logrus.FieldLogger, task models.RunTaskInfo) (bool, error) {
	loadCertsOnce.Do(func() {
		if err := loadCertMp(logger); err != nil {
			logger.WithError(err).Error("Load certs from local failed")
		} else {
			logger.Info("Load certs from local successd")
		}
	})

	fields := strings.SplitN(task.Signature, "#", 3)
	if len(fields) != 3 {
		return false, ErrorUnknownSignatureFormat
	}
	if fields[0] == "" || fields[1] == "" || fields[2] == "" {
		return false, ErrorUnknownSignatureFormat
	}
	signVer, err := strconv.Atoi(fields[0])
	if err != nil {
		return false, ErrorUnknownSignatureFormat
	}
	keypairVer, err := strconv.Atoi(fields[1])
	if err != nil {
		return false, ErrorUnknownSignatureFormat
	}
	signature := fields[2]
	signatureByte, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, err
	}

	key := fmt.Sprintf("%d-%d", signVer, keypairVer)
	var c *Cert
	if value, ok := certs_.Load(key); ok {
		c, _ = value.(*Cert)
	} else {
		c, err = updateCertsMp(logger, key, signVer, keypairVer)
		if err != nil {
			return false, err
		}
	}

	var dataList []string
	for _, field := range c.SignatureFields {
		switch field {
		case "userId":
			dataList = append(dataList, task.UserId)
		case "instanceId":
			instanceId := util.GetInstanceId()
			if instanceId == "unknown" {
				return false, ErrorUnknownInstanceId
			}
			dataList = append(dataList, instanceId)
		case "commandContent":
			dataList = append(dataList, task.Content)
		}
	}
	data := strings.Join(dataList, "#")
	logger.Info("dataList: ", data)

	switch c.Algorithm {
	case algorithm_sha256withrsa:
		return verifyWithSha256withrsa(c.PublicKeyRsa, data, signatureByte)
	case algorithm_sm3withsm2:
		return verifyWithSm3withsm2(c.PublicKeySm2, data, signatureByte)
	default:
		return false, ErrorUnknownSignAlgorithm
	}
}

// load certificates from cache file
func loadCertMp(logger logrus.FieldLogger) error {
	certsMpLock.Lock()
	defer certsMpLock.Unlock()

	certsMp = make(map[string]*Cert)
	crossVersionConfDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		return err
	}
	certFile := filepath.Join(crossVersionConfDir, certFileName)
	content, err := os.ReadFile(certFile)
	if err != nil {
		return err
	}
	certs := map[string]*Cert{}
	if err = json.Unmarshal(content, &certs); err != nil {
		return err
	}
	for k, c := range certs {
		if err := c.parsePublicKey(); err != nil {
			logger.WithError(err).Errorf("Parse public key[%d-%d] failed", c.SignatureVersion, c.KeypairVersion)
			continue
		}
		certs_.Store(k, c)
		certsMp[k] = c
	}
	return nil
}

// pull new certificate from online and update certificates cache file
func updateCertsMp(logger logrus.FieldLogger, key string, signVer int, keypairVer int) (*Cert, error) {
	certsMpLock.Lock()
	defer certsMpLock.Unlock()

	var c *Cert
	var ok bool
	var err error
	if c, ok = certsMp[key]; !ok {
		if key, c, err = pullCert(signVer, keypairVer); err != nil {
			return nil, err
		}
		certsMp[key] = c
		certs_.Store(key, c)
		if err := storeCertMp(); err != nil {
			logger.WithError(err).Error("Store cert to local failed")
		}
	}
	return c, nil
}

func pullCert(signVer, keypairVer int) (string, *Cert, error) {
	url := util.GetSignCertService()
	url += fmt.Sprintf("?signatureVersion=%d&keypairVersion=%d", signVer, keypairVer)
	err, resp := util.HttpGet(url)
	if err != nil {
		return "", nil, err
	}
	cert := &Cert{}
	if err = json.Unmarshal([]byte(resp), &cert); err != nil {
		return "", nil, err
	}
	if err := cert.parsePublicKey(); err != nil {
		return "", nil, err
	}
	key := fmt.Sprintf("%d-%d", cert.SignatureVersion, cert.KeypairVersion)
	return key, cert, nil
}

func storeCertMp() error {
	crossVersionConfDir, err := pathutil.GetCrossVersionConfigPath()
	if err != nil {
		return err
	}
	certFile := filepath.Join(crossVersionConfDir, certFileName)
	content, err := json.Marshal(certsMp)
	if err != nil {
		return err
	}
	return os.WriteFile(certFile, content, 0644)
}

func verifyWithSha256withrsa(pk *rsa.PublicKey, data string, signature []byte) (bool, error) {
	// Hash data before verifying signature.
	msgHash := crypto.SHA256.New()
	if _, err := msgHash.Write([]byte(data)); err != nil {
		return false, err
	}
	msgHashSum := msgHash.Sum(nil)

	err := rsa.VerifyPKCS1v15(pk, crypto.SHA256, msgHashSum, signature)
	return err == nil, err
}

func verifyWithSm3withsm2(pk *sm2.PublicKey, data string, signature []byte) (bool, error) {
	return pk.Verify([]byte(data), signature), nil
}
