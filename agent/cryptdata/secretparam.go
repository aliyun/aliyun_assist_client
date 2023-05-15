package cryptdata

import (
	"errors"
	"sync"
	"time"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

var (
	secretParams_ sync.Map
)

func CreateSecretParam(keyPairId, secretName string, timeout int64, encrypted []byte) (*ParamInfo, error) {
	decrypted, err := DecryptWithRsa(keyPairId, encrypted)
	if err != nil {
		return nil, err
	}
	timestamp := time.Now().Unix()
	if secretName == "" {
		secretName = util.ComputeStrMd5(fmt.Sprint(timestamp, decrypted))
	}
	param := &secretParam{
		SecretName: secretName,
		PlainText: string(decrypted),
		CreatedTimestamp: timestamp,
		ExpiredTimestamp: timestamp + timeout,
	}
	if err := storeParam(param.SecretName, param); err != nil {
		return nil, err
	}
	paramInfo := &ParamInfo{
		SecretName: secretName,
		CreatedTimestamp: timestamp,
		ExpiredTimestamp: timestamp + timeout,
	}
	return paramInfo, nil
}

func GetSecretParam(secretName string) (string, error) {
	param, err := loadParam(secretName)
	if err != nil {
		return "", err
	}
	return param.PlainText, nil
}

func clearExpiredParam() {
	ps := getParams()
	now := time.Now().Unix()
	for _, p := range ps {
		if p.ExpiredTimestamp <= now {
			log.GetLogger().Infof("SecretParam[%s] has expired for %d second, so delete it", p.SecretName, now-p.ExpiredTimestamp)
			secretParams_.Delete(p.SecretName)
		}
	}
}

func loadParam(name string) (*secretParam, error) {
	if value, ok := secretParams_.Load(name); !ok {
		return nil, ErrParamNotExist
	} else {
		param, ok := value.(*secretParam)
		if !ok {
			return nil, errors.New("Type convert failed")
		}
		now := time.Now().Unix()
		if param.ExpiredTimestamp < now {
			log.GetLogger().Infof("SecretParam[%s] has expired for %d second, so delete it", param.SecretName, now-param.ExpiredTimestamp)
			secretParams_.Delete(name)
			return nil, ErrParamNotExist
		}
		return param, nil
	}
}

func getParams() []*secretParam {
	params := []*secretParam{}
	secretParams_.Range(func(k, v interface{}) bool {
		if param, ok := v.(*secretParam); ok {
			params = append(params, param)
		}
		return true
	})
	return params
}

func storeParam(name string, param *secretParam) error {
	// if param exists, it will be overwrite
	secretParams_.Store(name, param)
	return nil
}