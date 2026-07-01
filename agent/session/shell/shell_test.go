package shell

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/encryptionutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	gomonkey "github.com/agiledragon/gomonkey/v2"
)

func TestNewShellPlugin(t *testing.T) {
	var logger *logrus.Entry
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Infof", func(_ *logrus.Entry, _ string, _ ...interface{}) {}).Reset()

	id := "testId"
	cmdContent := "testCmdContent"
	username := "testUsername"
	passwordName := "testPasswordName"
	flowLimit := 200 * 8 * 1024
	encryptionOptions := models.EncryptionOptions{
		Enabled:  true,
		Mode:     "Kms",
		KMSKeyId: "testKMSKeyId",
	}

	expectedShellPlugin := &ShellPlugin{}
	expectedShellPlugin.id = id
	expectedShellPlugin.cmdContent = cmdContent
	expectedShellPlugin.username = username
	expectedShellPlugin.passwordName = passwordName
	expectedShellPlugin.keyExchangeState = WaitClientHello
	expectedShellPlugin.encryptionOptions = encryptionOptions
	expectedShellPlugin.sendInterval = 1000 / (flowLimit / 8 / sendPackageSize)

	gotShellPlugin := NewShellPlugin(id, cmdContent, username, passwordName, flowLimit, encryptionOptions)

	assert.Equal(t, expectedShellPlugin.id, gotShellPlugin.id)
	assert.Equal(t, expectedShellPlugin.cmdContent, gotShellPlugin.cmdContent)
	assert.Equal(t, expectedShellPlugin.username, gotShellPlugin.username)
	assert.Equal(t, expectedShellPlugin.passwordName, gotShellPlugin.passwordName)
	assert.Equal(t, expectedShellPlugin.sendInterval, gotShellPlugin.sendInterval)
	assert.Equal(t, expectedShellPlugin.encryptionOptions, gotShellPlugin.encryptionOptions)
	assert.Equal(t, expectedShellPlugin.keyExchangeState, gotShellPlugin.keyExchangeState)
}

func TestExecute(t *testing.T) {
	encryptionOptions := models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"}
	p := NewShellPlugin("", "", "", "", 200*8*1024, encryptionOptions)
	dataChannel := &channel.SessionChannel{}
	cancelFlag := util.NewChanneledCancelFlag()

	var logger *logrus.Entry
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Infoln", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Errorln", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyFunc(time.After, func(_ time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}).Reset()
	// defer gomonkey.ApplyMethod(reflect.TypeOf(p), "stop", func(_ *ShellPlugin) error { return nil }).Reset()

	expectedSessionRes := sessionresult.NewKeyExchangeFailedError(fmt.Errorf("key exchange timeout"))
	gotSessionRes := p.Execute(dataChannel, cancelFlag)

	assert.Equal(t, expectedSessionRes.Code, gotSessionRes.Code)
	assert.Equal(t, expectedSessionRes.Info, gotSessionRes.Info)
}

func TestProcessStdoutData(t *testing.T) {
	encryptionOptions := models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"}
	p := NewShellPlugin("", "", "", "", 200*8*1024, encryptionOptions)
	p.exitCtx, p.exitFunc = context.WithCancelCause(context.Background())
	p.dataChannel = &channel.SessionChannel{}
	p.keyExchangeState = KeyExchangeSuccess
	stdoutBytes := []byte("test")
	stdoutBytesLen := len(stdoutBytes)
	unprocessedBuf := bytes.Buffer{}

	var logger *logrus.Entry
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Infoln", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Infof", func(_ *logrus.Entry, _ string, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Errorln", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Errorf", func(_ *logrus.Entry, _ string, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Error", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyFunc(encryptionutil.AESEncrypt, func(_ cipher.Block, _ []byte, outputStreamData []byte) []byte { return outputStreamData }).Reset()
	var dataChannel *channel.SessionChannel
	defer gomonkey.ApplyMethod(reflect.TypeOf(dataChannel), "SendKeyExchangeMessage", func(_ *channel.SessionChannel, payload []byte) error {
		return nil
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(dataChannel), "SendStreamDataMessage", func(_ *channel.SessionChannel, payload []byte) error {
		return nil
	}).Reset()

	_, err := p.processStdoutData(stdoutBytes, stdoutBytesLen, unprocessedBuf)

	assert.Nil(t, err)
}

func TestInputStreamMessageHandler(t *testing.T) {
	encryptionOptions := models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"}
	p := NewShellPlugin("", "", "", "", 200*8*1024, encryptionOptions)
	message := message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: 1, Payload: []byte{ClientHello}}

	var logger *logrus.Entry
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Infoln", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Errorln", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Error", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()
	defer gomonkey.ApplyFunc(time.After, func(_ time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	}).Reset()

	err := p.InputStreamMessageHandler(message)
	assert.Nil(t, err)
}

func TestKeyExchangeInInputStreamMessageHandler(t *testing.T) {
	kmsClientHelloMsg := ClientHelloMsg{
		CipherSuite: "AXT_KMS_WITH_AES_256_CBC",
		EncryptIV:   "AAAAAAAAAAAAAAAAAAAAAA==",
	}
	kmsClientHelloMsgBytes, _ := json.Marshal(kmsClientHelloMsg)
	kmsClientHelloMsgBytes = append([]byte{ClientHello}, kmsClientHelloMsgBytes...)
	kmsClientHelloMsgBytesLen := len(kmsClientHelloMsgBytes)

	ecdheClientHelloMsg := ClientHelloMsg{
		CipherSuite: "AXT_ECDHE_WITH_AES_256_CBC",
		EncryptIV:   "AAAAAAAAAAAAAAAAAAAAAA==",
	}
	ecdheClientHelloMsgBytes, _ := json.Marshal(ecdheClientHelloMsg)
	ecdheClientHelloMsgBytes = append([]byte{ClientHello}, ecdheClientHelloMsgBytes...)
	ecdheClientHelloMsgBytesLen := len(ecdheClientHelloMsgBytes)

	invalidClientHelloMsg := ClientHelloMsg{
		CipherSuite: "testCipherSuite",
		EncryptIV:   "AAAAAAAAAAAAAAAAAAAAAA==",
	}
	invalidClientHelloMsgBytes, _ := json.Marshal(invalidClientHelloMsg)
	invalidClientHelloMsgBytes = append([]byte{ClientHello}, invalidClientHelloMsgBytes...)
	invalidClientHelloMsgBytesLen := len(invalidClientHelloMsgBytes)

	kmsClientKeyExchangeMsg := KMSClientKeyExchangeMsg{
		KMSCipherTextKey: "testCipherTextKey",
	}
	kmsClientKeyExchangeMsgBytes, _ := json.Marshal(kmsClientKeyExchangeMsg)
	kmsClientKeyExchangeMsgBytes = append([]byte{ClientKeyExchange}, kmsClientKeyExchangeMsgBytes...)
	kmsClientKeyExchangeMsgBytesLen := len(kmsClientKeyExchangeMsgBytes)

	ecdheClientKeyExchangeMsg := ECDHEClientKeyExchangeMsg{
		ClientPub: "testClientPub",
		Signature: "testSignature",
	}
	ecdheClientKeyExchangeMsgBytes, _ := json.Marshal(ecdheClientKeyExchangeMsg)
	ecdheClientKeyExchangeMsgBytes = append([]byte{ClientKeyExchange}, ecdheClientKeyExchangeMsgBytes...)
	ecdheClientKeyExchangeMsgBytesLen := len(ecdheClientKeyExchangeMsgBytes)

	var dataChannel *channel.SessionChannel
	dataChannelGuard := gomonkey.ApplyMethod(reflect.TypeOf(dataChannel), "SendKeyExchangeMessage", func(_ *channel.SessionChannel, payload []byte) error {
		return nil
	})
	defer dataChannelGuard.Reset()

	timeGuard := gomonkey.ApplyFunc(time.After, func(d time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Now()
		return ch
	})
	defer timeGuard.Reset()

	kmsDecryptGuard := gomonkey.ApplyFunc(encryptionutil.KMSDecrypt, func(_ logrus.FieldLogger, _ string) (string, error) { return "1234567890123456", nil })
	defer kmsDecryptGuard.Reset()

	var shellPlugin *ShellPlugin
	initBlockGuard := gomonkey.ApplyMethod(reflect.TypeOf(shellPlugin), "InitBlock", func(_ *ShellPlugin, _ []byte) error { return nil })
	defer initBlockGuard.Reset()

	var encoding *base64.Encoding
	base64DecodeGuard := gomonkey.ApplyMethod(reflect.TypeOf(encoding), "DecodeString", func(_ *base64.Encoding, _ string) ([]byte, error) { return []byte{}, nil })
	defer base64DecodeGuard.Reset()

	generateSharedSecretGuard := gomonkey.ApplyFunc(encryptionutil.GenerateSharedSecret, func(_ *ecdh.PrivateKey, _ []byte) ([]byte, error) { return []byte("1234567890123456"), nil })
	defer generateSharedSecretGuard.Reset()

	var logger *logrus.Entry
	defer gomonkey.ApplyMethod(reflect.TypeOf(logger), "Error", func(_ *logrus.Entry, _ ...interface{}) {}).Reset()

	// kmsClientKeyExchangeMsg := KMSClientKeyExchangeMsg{
	// 	KMSCipherTextKey: "",
	// }
	// kmsClientKeyExchangeMsgBytes, _ := json.Marshal(kmsClientKeyExchangeMsg)
	// kmsClientKeyExchangeMsgBytes = append([]byte{ClientKeyExchange}, kmsClientKeyExchangeMsgBytes...)
	// kmsClientKeyExchangeMsgBytesLen := len(kmsClientKeyExchangeMsgBytes)

	tests := []struct {
		name                     string
		encryptionOptions        models.EncryptionOptions
		initialKeyExchangeState  int
		cipherSuite              string
		inputStreamMessage       message.Message
		expectedKeyExchangeState int
		expectedError            error
	}{
		{
			name:                     "KeyExchangeMessage.NotEnabledErr",
			encryptionOptions:        models.EncryptionOptions{Enabled: false},
			initialKeyExchangeState:  WaitClientHello,
			cipherSuite:              "AXT_KMS_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: 1, Payload: []byte{ClientHello}},
			expectedKeyExchangeState: WaitClientHello,
			expectedError:            fmt.Errorf("key exchange is not enabled"),
		},
		{
			name:                     "KeyExchangeMessage.KMS.ClientHello.Success",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  WaitClientHello,
			cipherSuite:              "AXT_KMS_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(kmsClientHelloMsgBytesLen), Payload: kmsClientHelloMsgBytes},
			expectedKeyExchangeState: SendAgentHello,
			expectedError:            nil,
		},
		{
			name:                     "KeyExchangeMessage.ECDHE.ClientHello.Success",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Auto", KMSKeyId: "test"},
			initialKeyExchangeState:  WaitClientHello,
			cipherSuite:              "AXT_ECDHE_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(ecdheClientHelloMsgBytesLen), Payload: ecdheClientHelloMsgBytes},
			expectedKeyExchangeState: SendAgentHello,
			expectedError:            nil,
		},
		{
			name:                     "KeyExchangeMessage.InvalidCipherSuiteError",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  WaitClientHello,
			cipherSuite:              "testCipherSuite",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(invalidClientHelloMsgBytesLen), Payload: invalidClientHelloMsgBytes},
			expectedKeyExchangeState: RecClientHello,
			expectedError:            fmt.Errorf("unknown CipherSuite in client hello message: testCipherSuite"),
		},
		{
			name:                     "KeyExchangeMessage.KMS.ClientKeyExchange.Success",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  SendAgentHello,
			cipherSuite:              "AXT_KMS_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(kmsClientKeyExchangeMsgBytesLen), Payload: kmsClientKeyExchangeMsgBytes},
			expectedKeyExchangeState: KeyExchangeSuccess,
			expectedError:            nil,
		},
		{
			name:                     "KeyExchangeMessage.ECDHE.ClientKeyExchange.Success",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Auto", KMSKeyId: "test"},
			initialKeyExchangeState:  SendAgentHello,
			cipherSuite:              "AXT_ECDHE_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(ecdheClientKeyExchangeMsgBytesLen), Payload: ecdheClientKeyExchangeMsgBytes},
			expectedKeyExchangeState: KeyExchangeSuccess,
			expectedError:            nil,
		},
		{
			name:                     "KeyExchangeMessage.ClientHello.InvalidActionTypeErr",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  WaitClientHello,
			cipherSuite:              "AXT_KMS_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(kmsClientHelloMsgBytesLen), Payload: append([]byte{ClientKeyExchange}, kmsClientHelloMsgBytes[1:]...)},
			expectedKeyExchangeState: WaitClientHello,
			expectedError:            fmt.Errorf("expecting ClientHello(0) while receiving actionType: %v", ClientKeyExchange),
		},
		{
			name:                     "KeyExchangeMessage.ClientKeyExchange.InvalidActionTypeErr",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  SendAgentHello,
			cipherSuite:              "AXT_KMS_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(kmsClientKeyExchangeMsgBytesLen), Payload: append([]byte{ClientHello}, kmsClientKeyExchangeMsgBytes[1:]...)},
			expectedKeyExchangeState: SendAgentHello,
			expectedError:            fmt.Errorf("expecting ClientKeyExchange(2) while receiving actionType: %v", ClientHello),
		},
		{
			name:                     "KeyExchangeMessage.ClientKeyExchange.InvalidStateErr",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  RecClientHello,
			cipherSuite:              "AXT_KMS_WITH_AES_256_CBC",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(kmsClientKeyExchangeMsgBytesLen), Payload: kmsClientKeyExchangeMsgBytes},
			expectedKeyExchangeState: RecClientHello,
			expectedError:            fmt.Errorf("receive KeyExchangeMessage in invalid key exchange state: %d", RecClientHello),
		},
		{
			name:                     "KeyExchangeMessage.ClientKeyExchange.InvalidCipherSuiteErr",
			encryptionOptions:        models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"},
			initialKeyExchangeState:  SendAgentHello,
			cipherSuite:              "testCipherSuite",
			inputStreamMessage:       message.Message{MessageType: message.KeyExchangeMessage, PayloadLength: uint32(kmsClientKeyExchangeMsgBytesLen), Payload: kmsClientKeyExchangeMsgBytes},
			expectedKeyExchangeState: SendAgentHello,
			expectedError:            fmt.Errorf("invalid ShellPlugin.cipherSuite: testCipherSuite"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewShellPlugin("", "", "", "", 200*8*1024, tt.encryptionOptions)
			p.exitCtx, p.exitFunc = context.WithCancelCause(context.Background())
			p.dataChannel = &channel.SessionChannel{}
			p.keyExchangeState = tt.initialKeyExchangeState
			p.cipherSuite = tt.cipherSuite
			close(p.initDone)

			err := p.InputStreamMessageHandler(tt.inputStreamMessage)
			keyExchangeState := p.keyExchangeState

			assert.Equal(t, tt.expectedError, err)
			assert.Equal(t, tt.expectedKeyExchangeState, keyExchangeState)
		})
	}
}

func TestSendKeyExchangeErrorMessage(t *testing.T) {
	encryptionOptions := models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"}
	p := NewShellPlugin("", "", "", "", 200*8*1024, encryptionOptions)
	p.dataChannel = &channel.SessionChannel{}
	message := "testMessage"
	code := "testCode"

	var dataChannel *channel.SessionChannel
	defer gomonkey.ApplyMethod(reflect.TypeOf(dataChannel), "SendKeyExchangeMessage", func(_ *channel.SessionChannel, _ []byte) error { return nil }).Reset()

	err := p.sendKeyExchangeErrorMessage(message, code)
	assert.Nil(t, err)
}

func TestInitBlock(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		expectedErr error
	}{
		{
			name:        "InitBlock.Success",
			key:         []byte("1234567890123456"),
			expectedErr: nil,
		},
		{
			name:        "InitBlock.NewCipherError",
			key:         []byte(""),
			expectedErr: aes.KeySizeError(0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewShellPlugin("", "", "", "", 200*8*1024, models.EncryptionOptions{Enabled: true, Mode: "Kms", KMSKeyId: "test"})
			err := p.InitBlock(tt.key)
			assert.Equal(t, tt.expectedErr, err)
		})
	}
}
