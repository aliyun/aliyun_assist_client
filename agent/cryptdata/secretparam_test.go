package cryptdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSecretParam(t *testing.T) {
	var (
		paramName = "pname"
		paramValue = "pvalue"
		timeoutSecond = 1
	)

	key, err := GenRsaKey("key", 120)
	defer RemoveRsaKey(key.Id)
	assert.Nil(t, err)
	encryptedParamValue, err := EncryptWithRsa(key.Id, paramValue)
	assert.Nil(t, err)
	p, err := CreateSecretParam(key.Id, paramName, int64(timeoutSecond), encryptedParamValue, nil)
	assert.Nil(t, err)
	assert.Equal(t, timeoutSecond, int(p.ExpiredTimestamp-p.CreatedTimestamp))
	assert.Equal(t, paramName, p.SecretName)
	value, err := GetSecretParamValue(p.SecretName)
	assert.Nil(t, err)
	assert.Equal(t, paramValue, value.SecretValue)
	time.Sleep(time.Duration(timeoutSecond+1) * time.Second)
	_, err = GetSecretParamValue(p.SecretName)
	assert.ErrorIs(t, err, ErrParamNotExist)
}
