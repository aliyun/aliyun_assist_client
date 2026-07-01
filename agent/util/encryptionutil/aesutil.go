package encryptionutil

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

func AESEncrypt(block cipher.Block, iv []byte, plainData []byte) []byte {
	cbcEncrypter := cipher.NewCBCEncrypter(block, iv)
	content := PKCS7Padding(plainData, aes.BlockSize)
	encryptedData := make([]byte, len(content))
	cbcEncrypter.CryptBlocks(encryptedData, content)
	return encryptedData
}

func AESDecrypt(block cipher.Block, iv []byte, encryptedData []byte) []byte {
	cbcDecrypter := cipher.NewCBCDecrypter(block, iv)
	decryptedData := make([]byte, len(encryptedData))
	cbcDecrypter.CryptBlocks(decryptedData, encryptedData)
	return PKCS7Trimming(decryptedData)
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7Trimming(encrypt []byte) []byte {
	length := len(encrypt)
	if length == 0 {
		return encrypt
	}
	padding := encrypt[len(encrypt)-1]
	return encrypt[:len(encrypt)-int(padding)]
}
