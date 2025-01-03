package cryptdata

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
	rd "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	plainText = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ 0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~ "
)

func TestAutoCleanKey(t *testing.T) {
	cases := []struct {
		KeyId         string
		TimeoutSecond int

		CheckTimeout bool
	}{
		{
			KeyId:         "k_timeout",
			TimeoutSecond: 1,
			CheckTimeout:  true,
		},
		{
			KeyId:         "k_normal",
			TimeoutSecond: 120,
			CheckTimeout:  false,
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

func TestVerify(t *testing.T) {
	keyId := "test-key"
	rawData := "This is test text!"
	_, err := GenRsaKey(keyId, 60)
	assert.Nil(t, err)

	signature, err := SignData(keyId, rawData)
	assert.Nil(t, err)
	signatureEncoded := base64.StdEncoding.EncodeToString(signature)
	signature, err = base64.StdEncoding.DecodeString(signatureEncoded)
	assert.Nil(t, err)

	valid, err := VerifySignature(keyId, rawData, signature)
	assert.Nil(t, err)
	assert.True(t, valid)

	signature = append(signature, 'a')
	valid, err = VerifySignature(keyId, rawData, signature)
	assert.Nil(t, err)
	assert.False(t, valid)
}

func TestEncryptAndDecryptLongData(t *testing.T) {
	keyInfo, err := GenRsaKey("", 60)
	assert.Nil(t, err)
	testLens := []int{}
	for i := 0; i < 1024; i += 1 {
		testLens = append(testLens, i)
	}
	testLens = append(testLens, []int{1024*5 + 3, 1024*10 + 3, 1024*100 + 3, 1024*1024 + 3}...)
	for _, i := range testLens {
		plainText := generateRandom(i)
		aesKey, encrypted, err := encryptWithAes(plainText)
		assert.Nil(t, err)
		encryptedKey, err := EncryptWithRsa(keyInfo.Id, string(aesKey))
		assert.Nil(t, err)

		// fmt.Println(string(plainText), base64.StdEncoding.EncodeToString(encryptedKey), base64.StdEncoding.EncodeToString(encrypted))
		// fmt.Println(len(plainText), len(encryptedKey), len(encrypted))

		aesKey_, err := DecryptWithRsa(keyInfo.Id, encryptedKey)
		assert.Nil(t, err)
		assert.Equal(t, aesKey, aesKey_)
		plainText_, err := decryptWithAes(encrypted, aesKey)
		assert.Nil(t, err)
		assert.Equal(t, plainText, plainText_)
	}
}

func encryptWithAes(plainText []byte) (key []byte, encrypted []byte, err error) {
	key = generateRandom(32)
	block, err := aes.NewCipher(key)
	if err != nil {
		return
	}

	blockSize := block.BlockSize()
	paddedText := pkcs7Padding(plainText, block.BlockSize())

	encrypted = make([]byte, blockSize+len(paddedText))
	iv := encrypted[:blockSize] // 初始化向量
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(encrypted[blockSize:], paddedText)

	return
}

func pkcs7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func generateRandom(length int) []byte {
	if length == 0 {
		return []byte{}
	}
	const charset = "!@#$%^&()_+=*abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rd.Intn(len(charset))]
	}
	return b
}
