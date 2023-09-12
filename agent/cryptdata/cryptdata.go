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

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var (
	keyPairs_ sync.Map

	ErrKeyIdNotExist   = errors.New("Key id not exist")
	ErrKeyIdDuplicated = errors.New("Key id is duplicated")
	ErrParamNotExist   = errors.New("Secret param not exist")
)

const (
	ERR_OTHER_CODE = 1
	// Error for 'agent not support' is 110
	ERR_KEYID_NOTEXIST_CODE   = 111
	ERR_KEYID_DUPLICATED_CODE = 112
	ERR_PARAM_NOTEXIST_CODE   = 113

	// length limit of plaintext to encrypt is 190byte, see https://crypto.stackexchange.com/questions/42097/what-is-the-maximum-size-of-the-plaintext-message-for-rsa-oaep
	LIMIT_PLAINTEXT_LEN = 190
)

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
	publicBlock := pem.Block{Type: "PUBLIC KEY", Bytes: X509PublicKey}
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

func RemoveRsaKey(keyId string) error {
	if k, _ := loadKey(keyId); k == nil {
		return ErrKeyIdNotExist
	}
	deleteKey(keyId)
	return nil
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
			log.GetLogger().Infof("KeyPair[%s] has expired for %d second, so delete it", k.Id, now-k.ExpiredTimestamp)
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
			log.GetLogger().Infof("KeyPair[%s] has expired for %d second, so delete it", privateKey.Id, now-privateKey.ExpiredTimestamp)
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

func deleteKey(keyId string) {
	keyPairs_.Delete(keyId)
	log.GetLogger().Infof("KeyPair[%s] is actively deleted", keyId)
}

func ErrToCode(err error) int {
	if errors.Is(err, ErrKeyIdDuplicated) {
		return ERR_KEYID_DUPLICATED_CODE
	} else if errors.Is(err, ErrKeyIdNotExist) {
		return ERR_KEYID_NOTEXIST_CODE
	} else if errors.Is(err, ErrParamNotExist) {
		return ERR_PARAM_NOTEXIST_CODE
	}
	return ERR_OTHER_CODE
}
