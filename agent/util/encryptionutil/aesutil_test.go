package encryptionutil

import (
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAESEncrypt(t *testing.T) {
	// 准备测试密钥和IV
	key := []byte("1234567890123456") // 16字节密钥
	iv := []byte("1234567890123456")  // 16字节IV

	// 创建真实的AES加密器用于测试
	block, _ := aes.NewCipher(key)

	tests := []struct {
		name           string
		input          []byte
		expectedBase64 string
	}{
		{
			name:           "正常明文数据",
			input:          []byte("hello world"),
			expectedBase64: "bAx40eFUVf/hIxbaV8/GaQ==",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行加密
			got := AESEncrypt(block, iv, tt.input)
			gotBase64 := base64.StdEncoding.EncodeToString(got)
			// 验证相同输入产生相同输出
			expectedBase64 := tt.expectedBase64
			fmt.Printf("expectedBase64: %s, gotBase64: %s\n", expectedBase64, gotBase64)
			assert.Equal(t, expectedBase64, gotBase64)
		})
	}
}

func TestAESDecrypt(t *testing.T) {
	// 准备测试密钥和IV
	key := []byte("1234567890123456") // 16字节密钥
	iv := []byte("1234567890123456")  // 16字节IV

	// 创建真实的AES加密器用于测试
	block, _ := aes.NewCipher(key)

	inputBytes, _ := base64.StdEncoding.DecodeString("bAx40eFUVf/hIxbaV8/GaQ==")

	tests := []struct {
		name        string
		input       []byte
		expectedStr string
	}{
		{
			name:        "正常密文数据",
			input:       inputBytes,
			expectedStr: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 执行加密
			gotStr := string(AESDecrypt(block, iv, tt.input))
			// 验证相同输入产生相同输出
			fmt.Printf("expectedStr: %s, gotStr: %s\n", tt.expectedStr, gotStr)
			assert.Equal(t, tt.expectedStr, gotStr)
		})
	}
}
