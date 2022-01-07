package util

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"io"
)

func RsaSign(data string, keyBytes string) string {
	w := md5.New()
	io.WriteString(w, data)
	md5_byte := w.Sum(nil)
	value := RsaSignWithMD5(md5_byte, []byte(keyBytes))
	encodeString := base64.StdEncoding.EncodeToString(value)
	return encodeString
}

func RsaSignWithMD5(data []byte, keyBytes []byte) []byte {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		log.GetLogger().Errorln("private key error")
		return []byte{}
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.GetLogger().Errorln("ParsePKCS8PrivateKey err")
		return []byte{}
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.MD5, data)
	if err != nil {
		log.GetLogger().Errorln("Error from signing")
		return []byte{}
	}

	return signature
}