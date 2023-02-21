package cryptdata

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/util"
)

type rsaKeyPair struct {
	Id               string
	CreatedTimestamp int64
	ExpiredTimestamp int64
	PrivateKey       *rsa.PrivateKey
	PublicKey        string
}

type KeyInfo struct {
	Id               string `json:"id"`
	CreatedTimestamp int64  `json:"createdTimestamp"`
	ExpiredTimestamp int64  `json:"expiredTimestamp"`
	PublicKey        string `json:"publicKey"`
}

type KeyInfos []KeyInfo

func (k KeyInfos) Len() int {
	return len(k)
}
func (k KeyInfos) Less(i, j int) bool {
	return k[i].ExpiredTimestamp < k[j].ExpiredTimestamp
}
func (k KeyInfos) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

var (
	keyPairs_ sync.Map

	ErrKeyIdNotExist   = errors.New("Key id not exist")
	ErrKeyIdDuplicated = errors.New("Key id is duplicated")

	clearExpiredKeyTimer_    *time.Timer
	clearExpiredKeyInterval_ = 60
)

const (
	ERR_OTHER_CODE = 1
	// Error for 'agent not support' is 110
	ERR_KEYID_NOTEXIST_CODE   = 111
	ERR_KEYID_DUPLICATED_CODE = 112

	// length limit of plaintext to encrypt is 190byte, see https://crypto.stackexchange.com/questions/42097/what-is-the-maximum-size-of-the-plaintext-message-for-rsa-oaep
	LIMIT_PLAINTEXT_LEN = 190
)

func init() {
	go func() {
		clearExpiredKeyTimer_ = time.NewTimer(time.Duration(clearExpiredKeyInterval_) * time.Second)
		for {
			_ = <-clearExpiredKeyTimer_.C
			clearExpiredKey()
			clearExpiredKeyTimer_.Reset(time.Duration(clearExpiredKeyInterval_) * time.Second)
		}
	}()
}

func GenRsaKey(specifiedId string, timeout int) (*KeyInfo, error) {
	if specifiedId != "" {
		if k, _ := loadKey(specifiedId); k != nil {
			return nil, ErrKeyIdDuplicated
		}
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	publicKey := privateKey.PublicKey
	var X509PublicKey []byte
	X509PublicKey, err = x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		return nil, err
	}
	publicBlock := pem.Block{Type: "Public Key", Bytes: X509PublicKey}
	buf := bytes.NewBufferString("")
	if err = pem.Encode(buf, &publicBlock); err != nil {
		return nil, err
	}
	var keyId, publicKeyStr string
	publicKeyStr = buf.String()
	timestamp := time.Now().Unix()
	if specifiedId != "" {
		keyId = specifiedId
	} else {
		keyId = util.ComputeStrMd5(fmt.Sprint(timestamp, publicKeyStr))
	}
	keyPair := &rsaKeyPair{
		Id:               keyId,
		CreatedTimestamp: timestamp,
		ExpiredTimestamp: timestamp + int64(timeout),
		PrivateKey:       privateKey,
		PublicKey:        publicKeyStr,
	}
	if err = storeKey(keyId, keyPair); err != nil {
		return nil, err
	}
	keyInfo := &KeyInfo{
		Id:               keyId,
		CreatedTimestamp: timestamp,
		ExpiredTimestamp: timestamp + int64(timeout),
		PublicKey:        publicKeyStr,
	}
	return keyInfo, nil
}

func EncryptWithRsa(keyId, rawData string) ([]byte, error) {
	if privateKey, err := loadKey(keyId); err != nil {
		return nil, err
	} else {
		if encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, &privateKey.PrivateKey.PublicKey, []byte(rawData), nil); err != nil {
			return nil, err
		} else {
			return encrypted, nil
		}
	}
}

func DecryptWithRsa(keyId string, encrypted []byte) ([]byte, error) {
	if privateKey, err := loadKey(keyId); err != nil {
		return nil, err
	} else {
		if decrypted, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey.PrivateKey, encrypted, nil); err != nil {
			return nil, err
		} else {
			return decrypted, nil
		}
	}
}

func CheckKey(keyId string) (*KeyInfo, error) {
	if privateKey, err := loadKey(keyId); err != nil {
		return nil, err
	} else {
		keyInfo := &KeyInfo{
			Id:               privateKey.Id,
			CreatedTimestamp: privateKey.CreatedTimestamp,
			ExpiredTimestamp: privateKey.ExpiredTimestamp,
			PublicKey:        privateKey.PublicKey,
		}
		return keyInfo, nil
	}
}

func CheckKeyList() (keyList KeyInfos) {
	ks := getKeys()
	now := time.Now().Unix()
	for _, k := range ks {
		if k.ExpiredTimestamp <= now {
			continue
		}
		keyList = append(keyList, KeyInfo{
			Id:               k.Id,
			CreatedTimestamp: k.CreatedTimestamp,
			ExpiredTimestamp: k.ExpiredTimestamp,
			PublicKey:        k.PublicKey,
		})
	}
	sort.Sort(keyList)
	return
}

func clearExpiredKey() {
	ks := getKeys()
	now := time.Now().Unix()
	for _, k := range ks {
		if k.ExpiredTimestamp <= now {
			keyPairs_.Delete(k.Id)
		}
	}
}

func loadKey(keyId string) (*rsaKeyPair, error) {
	if value, ok := keyPairs_.Load(keyId); !ok {
		return nil, ErrKeyIdNotExist
	} else {
		privateKey, ok := value.(*rsaKeyPair)
		if !ok {
			return nil, errors.New("Type convert failed")
		}
		now := time.Now().Unix()
		if privateKey.ExpiredTimestamp < now {
			keyPairs_.Delete(keyId)
			return nil, ErrKeyIdNotExist
		}
		return privateKey, nil
	}
}

func getKeys() []*rsaKeyPair {
	keys := []*rsaKeyPair{}
	keyPairs_.Range(func(k, v interface{}) bool {
		if privateKey, ok := v.(*rsaKeyPair); ok {
			keys = append(keys, privateKey)
		}
		return true
	})
	return keys
}

func storeKey(keyId string, keyPair *rsaKeyPair) error {
	if _, ok := keyPairs_.LoadOrStore(keyId, keyPair); ok {
		return ErrKeyIdDuplicated
	}
	return nil
}

func ErrToCode(err error) int {
	if errors.Is(err, ErrKeyIdDuplicated) {
		return ERR_KEYID_DUPLICATED_CODE
	} else if errors.Is(err, ErrKeyIdNotExist) {
		return ERR_KEYID_NOTEXIST_CODE
	}
	return ERR_OTHER_CODE
}
