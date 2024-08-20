package cryptdata

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	plainText = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ 0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ "
)

func TestCryptData(t *testing.T) {
	cases := []struct{
		KeyId string
		TimeoutSecond int
		
		CheckTimeout bool
	}{
		{
			KeyId: "k_timeout",
			TimeoutSecond: 1,
			CheckTimeout: true,
		},
		{
			KeyId: "k_normal",
			TimeoutSecond: 120,
			CheckTimeout: false,
		},
	}

	// length of plainText need be 190 which is the max length allowed
	assert.Equal(t, 190, len(plainText))

	for _, c := range cases {
		k, err := GenRsaKey(c.KeyId, c.TimeoutSecond)
		assert.Nil(t, err)
		assert.Equal(t, c.TimeoutSecond, int(k.ExpiredTimestamp-k.CreatedTimestamp))
		assert.Equal(t, c.KeyId, k.Id)
		_, err = CheckKey(c.KeyId)
		assert.Nil(t, err)
		if c.CheckTimeout {
			time.Sleep(time.Duration(c.TimeoutSecond+1) * time.Second)
			_, err = EncryptWithRsa(c.KeyId, plainText)
			assert.ErrorIs(t, err, ErrKeyIdNotExist)
			_, err = CheckKey(c.KeyId)
			assert.ErrorIs(t, err, ErrKeyIdNotExist)
		} else {
			content, err := EncryptWithRsa(c.KeyId, plainText)
			assert.Nil(t, err)
			plainContent, err := DecryptWithRsa(c.KeyId, content)
			assert.Nil(t, err)
			assert.Equal(t, plainText, string(plainContent))

			err = RemoveRsaKey(c.KeyId)
			assert.Nil(t, err)
			_, err = CheckKey(c.KeyId)
			assert.ErrorIs(t, err, ErrKeyIdNotExist)
		}
	}
}