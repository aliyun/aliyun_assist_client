package shell

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"
	"unicode/utf8"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/channel"
	"github.com/aliyun/aliyun_assist_client/agent/session/message"
	"github.com/aliyun/aliyun_assist_client/agent/session/sessionresult"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/models"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/encryptionutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type ShellPluginBase struct {
	id           string
	stdin        *os.File
	stdout       *os.File
	cmdContent   string
	username     string
	passwordName string
	dataChannel  channel.ISessionChannel
	first_ws_col uint32
	first_ws_row uint32
	sendInterval int

	encryptionOptions models.EncryptionOptions
	keyExchangeDone   chan struct{} // 用于同步（1）伪终端创建（2）密钥协商完成，保证（1）在（2）之后发生
	keyExchangeState  int           // {0: 未收到ClientHello(未交换), 1: 已收到ClientHello, 2: 已发送AgentHello, 3:已收到ClientKeyExchange, 4: 已发送AgentKeyExchange(交换完成)}
	cipherSuite       string
	dataKey           string // 数据密钥
	iv                string
	block             cipher.Block
	initDone          chan struct{} // 用于同步（1）处理input消息（2）dataChannel和exitFunc初始化，保证（1）在（2）之后发生

	exitCtx  context.Context
	exitFunc context.CancelCauseFunc

	logger logrus.FieldLogger
}

type SizeData struct {
	Cols uint32 `json:"cols"`
	Rows uint32 `json:"rows"`
}

type ClientHelloMsg struct {
	CipherSuite string `json:"cipherSuite"`
	EncryptIV   string `json:"encryptIV"`
}

type AgentHelloMsg struct {
	ClientCertificate bool `json:"clientCertificate"`
}

type KMSClientKeyExchangeMsg struct {
	KMSCipherTextKey string `json:"kmsCipherTextKey"`
}

type ECDHEClientKeyExchangeMsg struct {
	ClientPub string `json:"clientPub"`
	Signature string `json:"signature"`
}

type KMSAgentKeyExchangeMsg struct{}

type ECDHEAgentKeyExchangeMsg struct {
	AgentPub string `json:"agentPub"`
}

type KeyExchangeErrorMsg struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	sendPackageSize     = 1024                                                   // 发送的payload大小上限，单位 B
	defaultSendSpeed    = 200                                                    // 默认的最大数据发送速率，单位 kbps
	defaultSendInterval = 1000 / (defaultSendSpeed * 1024 / 8 / sendPackageSize) // writeloop的循环间隔时间 单位ms
)

// 用于PortPlugin的keyExchangeState字段，标识密钥协商进度
const (
	WaitClientHello = iota
	RecClientHello
	SendAgentHello
	RecClientKeyExchange
	KeyExchangeSuccess
)

// 密钥协商过程中的actionType
const (
	ClientHello       = 0
	AgentHello        = 1
	ClientKeyExchange = 2
	AgentKeyExchange  = 3
	KeyExchangeError  = 11
)

const (
	KMSCipherSuite   = "AXT_KMS_WITH_AES_256_CBC"
	ECDHECipherSuite = "AXT_ECDHE_WITH_AES_256_CBC"
)

var isValidCipherSuite map[string]bool = map[string]bool{
	KMSCipherSuite:   true,
	ECDHECipherSuite: true,
}

func NewShellPlugin(id string, cmdContent string, username string, passwordName string, flowLimit int, encryptionOptions models.EncryptionOptions) *ShellPlugin {
	plugin := &ShellPlugin{}
	plugin.id = id
	plugin.cmdContent = cmdContent
	plugin.username = username
	plugin.passwordName = passwordName
	plugin.sendInterval = defaultSendInterval
	plugin.keyExchangeDone = make(chan struct{})
	plugin.keyExchangeState = WaitClientHello
	plugin.encryptionOptions = encryptionOptions
	plugin.initDone = make(chan struct{})
	plugin.logger = log.GetLogger().WithFields(logrus.Fields{
		"sessionType": "shell",
		"sessionId":   id,
	})
	if flowLimit > 0 {
		plugin.sendInterval = 1000 / (flowLimit / 8 / sendPackageSize)
	} else {
		flowLimit = defaultSendSpeed * 1024
	}
	plugin.logger.Infof("Init send speed, speed[%d]bps sendInterval[%d]ms\n", flowLimit, plugin.sendInterval)
	return plugin
}

func (p *ShellPlugin) Execute(dataChannel channel.ISessionChannel, cancelFlag util.CancelFlag) (sessionRes *sessionresult.SessionResult) {
	p.dataChannel = dataChannel
	// p.exitCtx and p.exitFunc are initialized in Execute() instead of NewPortPlugin() to ensure
	// that p.exitCtx can be released eventually.
	// Although InputStreamMessageHandler() may be called before Execute(), it will wait until
	// p.conn is successfully created before starting to process data (possibly calling p.exitFunc),
	// so p.exitFunc will not be called before it be initialized.
	p.exitCtx, p.exitFunc = context.WithCancelCause(context.Background())

	// 同步：在InputStreamMessageHandler处理输入之前完成dataChannel和exitFunc的初始化
	close(p.initDone)

	defer func() {
		p.logger.Infoln("stop in run ShellPlugin")
		if err := p.stop(); err != nil {
			p.logger.WithError(err).Error("Error occurred while closing pty")
		}

		if err := recover(); err != nil {
			p.logger.Errorf("Error occurred while executing plugin %s: \n%v", p.id, err)
			sessionRes = sessionresult.NewUnknownError(err)
		}
	}()
	// 同步等待密钥协商完成后，再开启伪终端
	if p.encryptionOptions.Enabled {
		select {
		case <-p.keyExchangeDone:
			// 密钥协商完成
			p.logger.Infoln("finish key exchange")
		case <-time.After(time.Second * 20):
			// 超时且密钥协商未完成
			p.logger.Errorln("key exchange timeout")
			sessionRes = sessionresult.NewKeyExchangeFailedError(fmt.Errorf("key exchange timeout"))
			return
		}
	}
	p.logger.Infoln("start pty")
	sessionRes = StartPty(p)
	if sessionRes != nil {
		p.logger.Errorln("Unable to start shell: ", sessionRes)
		return
	}
	p.logger.Infoln("start pty success")
	go func() {
		select {
		case <-cancelFlag.C():
			cancelState := cancelFlag.State()
			p.logger.Info("Cancel flag set to %v in session", cancelState)
			if cancelState == util.ErrerOccurred {
				_, err := cancelFlag.IsErrorOccurred()
				p.logger.WithError(err).Error("An error occurred in session")
				p.exitFunc(err)
			} else {
				p.logger.Info("Session is notified to be ended")
				p.exitFunc(sessionresult.NewNotified())
			}
		case <-p.exitCtx.Done():
			cancelFlag.Set(util.ShutDown)
		}
	}()

	go func() {
		p.writePump()
	}()
	p.logger.Infof("Plugin %s started", p.id)

	<-p.exitCtx.Done()
	contextErr := context.Cause(p.exitCtx)
	var ok bool
	if sessionRes, ok = contextErr.(*sessionresult.SessionResult); !ok {
		sessionRes = sessionresult.NewUnknownError(contextErr)
	}
	p.logger.Infoln("Plugin  done", p.id, sessionRes)

	if osutil.GetOsType() == osutil.OSLinux || osutil.GetOsType() == osutil.OSFreebsd {
		p.waitPid()
	}

	return
}

func (p *ShellPlugin) writePump() {
	defer func() {
		if err := recover(); err != nil {
			p.logger.Println("WritePump thread crashed with message: \n", err)
		}
	}()

	stdoutBytes := make([]byte, sendPackageSize)
	reader := bufio.NewReader(p.stdout)

	// Wait for all input commands to run.
	time.Sleep(time.Second)

	var unprocessedBuf bytes.Buffer

	for {
		select {
		case <-p.exitCtx.Done():
			return
		default:
			stdoutBytesLen, err := reader.Read(stdoutBytes)
			if err != nil {
				p.logger.Debugf("Failed to read from pty master: %s", err)
				p.exitFunc(sessionresult.NewOk("read from pty failed, " + err.Error() + "."))
				return
			}

			// unprocessedBuf contains incomplete utf8 encoded unicode bytes returned after processing of stdoutBytes
			var processErr *sessionresult.SessionResult
			if unprocessedBuf, processErr = p.processStdoutData(stdoutBytes, stdoutBytesLen, unprocessedBuf); processErr != nil {
				p.logger.Errorf("Error processing stdout data, %v", processErr)
				p.exitFunc(processErr)
				return
			}
			// Wait for stdout to process more data
			time.Sleep(time.Duration(p.sendInterval) * time.Millisecond)
		}
	}
}

// processStdoutData reads utf8 encoded unicode characters from stdoutBytes and sends it over websocket channel.
func (p *ShellPlugin) processStdoutData(
	stdoutBytes []byte,
	stdoutBytesLen int,
	unprocessedBuf bytes.Buffer) (bytes.Buffer, *sessionresult.SessionResult) {

	// append stdoutBytes to unprocessedBytes and then read rune from appended bytes to send it over websocket channel
	unprocessedBytes := unprocessedBuf.Bytes()
	unprocessedBytes = append(unprocessedBytes[:], stdoutBytes[:stdoutBytesLen]...)
	runeReader := bufio.NewReader(bytes.NewReader(unprocessedBytes))

	var processedBuf bytes.Buffer
	unprocessedBytesLen := len(unprocessedBytes)
	i := 0
	for i < unprocessedBytesLen {
		// read stdout bytes as utf8 encoded unicode character
		stdoutRune, stdoutRuneLen, err := runeReader.ReadRune()
		if err != nil {
			return processedBuf, sessionresult.NewProcessStdoutDataErrorError(err)
		}

		// Invalid utf8 encoded character results into RuneError.
		if stdoutRune == utf8.RuneError {

			// If invalid character is encountered within last 3 bytes of buffer (utf8 takes 1-4 bytes for a unicode character),
			// then break the loop and leave these bytes in unprocessed buffer for them to get processed later with more bytes returned by stdout.
			if unprocessedBytesLen-i < utf8.UTFMax {
				runeReader.UnreadRune()
				break
			}

			// If invalid character is encountered beyond last 3 bytes of buffer, then the character at ith position is invalid utf8 character.
			// Add invalid byte at ith position to processedBuf in such case and return to client to handle display of invalid character.
			processedBuf.Write(unprocessedBytes[i : i+1])
		} else {
			processedBuf.WriteRune(stdoutRune)
		}
		i += stdoutRuneLen
	}

	if p.dataChannel != nil {
		outputStreamData := processedBuf.Bytes()
		// p.logger.Infof("OutputStreamData (before encryption):", outputStreamData)
		if p.encryptionOptions.Enabled {
			// 加密通信
			// 检查密钥协商状态
			if p.keyExchangeState != KeyExchangeSuccess {
				p.logger.Errorf("Trying to output stream data before finishing key exchange, keyExchangeState: %v", p.keyExchangeState)
				// 发送密钥协商错误报文
				if sendErr := p.sendKeyExchangeErrorMessage("Trying to output stream data before finishing key exchange", "AgentOutputBeforeKeyExchange"); sendErr != nil {
					p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
				}
			} else {
				// AES加密
				outputStreamData = encryptionutil.AESEncrypt(p.block, []byte(p.iv), outputStreamData)
				// p.logger.Infof("OutputStreamData (after encryption):", outputStreamData)
			}
		}
		// 发送数据
		if err := p.dataChannel.SendStreamDataMessage(outputStreamData); err != nil {
			return processedBuf, sessionresult.NewSendingDataFailedError(err)
		}
	}

	// log.GetLogger().Println("data output: ", string(processedBuf.Bytes()))
	// return incomplete utf8 encoded unicode bytes to be processed with next batch of stdoutBytes
	unprocessedBuf.Reset()
	if i < unprocessedBytesLen {
		unprocessedBuf.Write(unprocessedBytes[i:unprocessedBytesLen])
	}
	return unprocessedBuf, nil
}

func (p *ShellPlugin) InputStreamMessageHandler(streamDataMessage message.Message) error {
	// 同步：在ShellPlugin完成Execute中dataChannel、exitFunc初始化后
	if p.dataChannel == nil || p.exitFunc == nil {
		select {
		case <-p.initDone:
			// 初始化完成
			p.logger.Infoln("finish p.dataChannel and p.exitFunc initialization")
		case <-time.After(time.Second * 10):
			// 超时且初始化未完成
			p.logger.Errorln("p.dataChannel and p.exitFunc initialization init timeout")
			return nil
		}
	}

	// InputStreamDataMessage消息的处理依赖于pty创建
	if streamDataMessage.MessageType == message.InputStreamDataMessage && (p.stdin == nil || p.stdout == nil) {
		// This is to handle scenario when cli/console starts sending size data but pty has not been started yet
		// Since packets are rejected, cli/console will resend these packets until pty starts successfully in separate thread
		p.logger.Error("Pty unavailable. Reject incoming message packet")
		return nil
	}

	switch streamDataMessage.MessageType {
	case message.InputStreamDataMessage:
		// p.logger.Infof("Input message received: ", streamDataMessage.Payload)
		// log.GetLogger().Traceln("Input message received: ", streamDataMessage.Payload)
		var payload []byte
		if p.encryptionOptions.Enabled {
			// 加密通信
			// 检查密钥协商状态
			if p.keyExchangeState != KeyExchangeSuccess {
				p.logger.Errorf("Receive InputStreamDataMessage before finishing key exchange, keyExchangeState: %v", p.keyExchangeState)
				// 发送密钥协商错误报文
				if sendErr := p.sendKeyExchangeErrorMessage("Receive InputStreamDataMessage before finishing key exchange", "RecvDataMessageBeforeKeyExchange"); sendErr != nil {
					p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
				}
				err := fmt.Errorf("receive InputStreamDataMessage before finishing key exchange, keyExchangeState: %v", p.keyExchangeState)
				return err
			}
			// 解密payload
			payload = encryptionutil.AESDecrypt(p.block, []byte(p.iv), streamDataMessage.Payload)
		} else {
			payload = streamDataMessage.Payload
		}
		// p.logger.Infof("Received InputStreamDataMessage: %v", payload)
		if err := p.onInputStreamData(payload); err != nil {
			return err
		}
	case message.SetSizeDataMessage:
		var size SizeData
		if err := json.Unmarshal(streamDataMessage.Payload, &size); err != nil {
			p.logger.WithError(err).Error("Invalid size message")
			return err
		}
		// log.GetLogger().Tracef("Resize data received: cols: %d, rows: %d", size.Cols, size.Rows)
		if err := p.SetSize(size.Cols, size.Rows); err != nil {
			p.logger.WithError(err).Error("Unable to set pty size")
			return err
		}
	case message.StatusDataMessage:
		if len(streamDataMessage.Payload) > 0 {
			code, err := message.BytesToIntU(streamDataMessage.Payload[0:1])
			if err == nil {
				switch code {
				case 7: // 设置agent的发送速率
					speed, err := message.BytesToIntU(streamDataMessage.Payload[1:]) // speed 单位是 bps
					if speed == 0 {
						break
					}
					if err != nil {
						p.logger.WithError(err).Error("Invalid flowLimit")
						return err
					}
					p.sendInterval = 1000 / (speed / 8 / sendPackageSize)
					p.logger.Infof("Set send speed, speed[%d]bps sendInterval[%d]ms\n", speed, p.sendInterval)
				case 5:
					p.logger.Info("Exit due to receiving a packet with a close status")
					p.exitFunc(sessionresult.NewOk("a close status packet was received."))
				}
			} else {
				p.logger.WithError(err).Errorf("Parse status code err")
			}
		}
	case message.CloseDataChannel:
		p.logger.Info("Exit due to receiving CloseDataChannel packet")
		p.exitFunc(sessionresult.NewOk("a CloseDataChannel packet was received."))
	case message.KeyExchangeMessage:
		// 若当前sessionTask未启用加密，则报错退出
		if !p.encryptionOptions.Enabled {
			p.logger.Errorln("Received KeyExchangeMessage packet while encryption is not enabled")
			err := fmt.Errorf("key exchange is not enabled")
			p.exitFunc(sessionresult.NewKeyExchangeFailedError(err))
			return err
		}
		// handle密钥协商报文
		if err := p.KeyExchangeMessageHandler(streamDataMessage.Payload); err != nil {
			p.exitFunc(sessionresult.NewKeyExchangeFailedError(err))
			return err
		}
	}
	return nil
}

func (p *ShellPlugin) KeyExchangeMessageHandler(payload []byte) error {
	// 解析actionType
	actionType, err := message.BytesToIntU(payload[0:1])
	if err != nil {
		p.logger.WithError(err).Error("Fail to parse actionType from payload in key exchange message")
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage("Fail to parse actionType from payload in key exchange message", "ActionTypeParseError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		return err
	}
	// 密钥协商
	switch p.keyExchangeState {
	case WaitClientHello:
		if actionType != ClientHello {
			p.logger.Errorf("Expecting ClientHello(0) while receiving actionType: %v", actionType)
			// 发送密钥协商错误报文
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Waiting for ClientHello(0) while received actionType=%v", actionType), "WrongActionTypeError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			err = fmt.Errorf("expecting ClientHello(0) while receiving actionType: %v", actionType)
			return err
		}
		// 收到ClientHello报文，开始密钥协商
		p.keyExchangeState = RecClientHello
		// 解析ClientHello报文内容
		var clientHelloMsg ClientHelloMsg
		if err := json.Unmarshal(payload[1:], &clientHelloMsg); err != nil {
			p.logger.WithError(err).Error("Invalid client hello message content")
			// 发送密钥协商错误报文
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Invalid client hello message content: %+v", clientHelloMsg), "ClientHelloParseError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			return err
		}
		// p.logger.Infof("Received ClientHello Message: %+v\n", clientHelloMsg)
		// 保存iv向量
		var ivByte []byte
		ivByte, err = base64.StdEncoding.DecodeString(clientHelloMsg.EncryptIV)
		if err != nil {
			p.logger.WithError(err).Error("Failed to decode iv with base64")
			// 发送密钥协商错误报文
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Failed to decode iv with base64: %+v", clientHelloMsg), "IVDecodeError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			return err
		}
		p.iv = string(ivByte)
		p.cipherSuite = clientHelloMsg.CipherSuite
		// clientHelloMsg内容检查
		isValid, exists := isValidCipherSuite[p.cipherSuite]
		if !exists {
			p.logger.Errorf("Unknown CipherSuite in client hello message: %v\n", clientHelloMsg.CipherSuite)
			// 发送密钥协商错误报文
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Unknown CipherSuite in client hello message: %+v", clientHelloMsg), "UnknownCipherSuiteError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			err := fmt.Errorf("unknown CipherSuite in client hello message: %v", clientHelloMsg.CipherSuite)
			return err
		} else if !isValid {
			p.logger.Errorf("Unsupported CipherSuite in client hello message: %v\n", clientHelloMsg.CipherSuite)
			// 发送密钥协商错误报文
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Unsupported CipherSuite in client hello message: %+v", clientHelloMsg), "UnsupportedCipherSuiteError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			err := fmt.Errorf("unsupported CipherSuite in client hello message: %v", clientHelloMsg.CipherSuite)
			return err
		}
		// 发送AgentHello报文
		agentHelloMsg := &AgentHelloMsg{ClientCertificate: false}
		agentHelloMsgContentBytes, err := json.Marshal(agentHelloMsg)
		if err != nil {
			p.logger.WithError(err).Error("Invalid AgentHelloMsg content")
			return err
		}
		// 添加actionType（content的第一个byte）
		agentHelloMsgBytes := append([]byte{byte(AgentHello)}, agentHelloMsgContentBytes...)
		if err := p.dataChannel.SendKeyExchangeMessage(agentHelloMsgBytes); err != nil {
			p.logger.WithError(err).Error("Unable to send agent hello message")
			return err
		}
		p.keyExchangeState = SendAgentHello
		// p.logger.Infof("Sent AgentHello Message: %+v\n", agentHelloMsg)
	case SendAgentHello:
		if actionType != ClientKeyExchange {
			p.logger.Errorf("Expecting ClientKeyExchange(2) while receiving actionType: %v", actionType)
			// 发送密钥协商错误报文
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Waiting for ClientKeyExchange(2) while received actionType=%v", actionType), "WrongActionTypeError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			err := fmt.Errorf("expecting ClientKeyExchange(2) while receiving actionType: %v", actionType)
			return err
		}
		switch p.cipherSuite {
		case KMSCipherSuite:
			if err := p.KMSClientKeyExchangeMessageHandler(payload); err != nil {
				p.logger.WithError(err).Error("Failed to handle KMSClientKeyExchangeMessage")
				return err
			}
		case ECDHECipherSuite:
			if err := p.ECDHEClientKeyExchangeMessageHandler(payload); err != nil {
				p.logger.WithError(err).Error("Failed to handle ECDHEClientKeyExchangeMessage")
				return err
			}
		default:
			// 发送密钥协商错误报文
			p.logger.Errorf("Invalid ShellPlugin.cipherSuite: %v", p.cipherSuite)
			if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Invalid ShellPlugin.cipherSuite: %v", p.cipherSuite), "InvalidCipherSuiteError"); sendErr != nil {
				p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
			}
			err := fmt.Errorf("invalid ShellPlugin.cipherSuite: %v", p.cipherSuite)
			return err
		}
		p.keyExchangeState = KeyExchangeSuccess
		// p.logger.Infof("Key exchange success, cipherSuite: %s", p.cipherSuite)
		close(p.keyExchangeDone)
	default:
		// 在RecClientHello、RecClientKeyExchange、KeyExchangeSuccess状态时，收到KeyExchangeMessage报文
		p.logger.Errorf("Receive KeyExchangeMessage in invalid key exchange state: %d\n", p.keyExchangeState)
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Receive KeyExchangeMessage in invalid key exchange state(%v)", p.keyExchangeState), "InvalidKeyExchangeStateError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		err = fmt.Errorf("receive KeyExchangeMessage in invalid key exchange state: %d", p.keyExchangeState)
		return err
	}
	return nil
}

func (p *ShellPlugin) KMSClientKeyExchangeMessageHandler(payload []byte) error {
	// 收到ClientKeyExchange报文，继续密钥协商流程
	p.keyExchangeState = RecClientKeyExchange
	// 解析ClientKeyExchange报文
	var kmsClientKeyExchangeMsg KMSClientKeyExchangeMsg
	if err := json.Unmarshal(payload[1:], &kmsClientKeyExchangeMsg); err != nil {
		p.logger.WithError(err).Error("Invalid KMS client key exchange message content")
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Invalid KMS client key exchange message content: %+v", kmsClientKeyExchangeMsg), "ClientKeyExchangeParseError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		return err
	}
	// p.logger.Infof("Received KMSClientKeyExchange Message: %+v\n", kmsClientKeyExchangeMsg)
	// 调用KMS的openapi对数据密钥进行解密
	cipherText := kmsClientKeyExchangeMsg.KMSCipherTextKey
	dataKey, err := encryptionutil.KMSDecrypt(p.logger, cipherText)
	if err != nil {
		p.logger.WithError(err).Error("KMS decrypt failed")
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("KMS decrypt failed: %s", err), "KMSDecryptError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		return err
	}
	// 存储dataKey
	p.dataKey = dataKey
	// 创建cbc对象用于AES加密
	if err := p.InitBlock([]byte(p.dataKey)); err != nil {
		p.logger.WithError(err).Errorf("Failed to init CBC, dataKey='%v'", p.dataKey)
		return err
	}
	// 创建AgentKeyExchange报文
	agentKeyExchangeMsg := &KMSAgentKeyExchangeMsg{}
	agentKeyExchangeMsgContentBytes, err := json.Marshal(agentKeyExchangeMsg)
	if err != nil {
		p.logger.WithError(err).Error("Invalid AgentKeyExchangeMsgBytes content")
		return err
	}
	// 添加actionType（content的第一个byte）
	agentKeyExchangeMsgBytes := append([]byte{byte(AgentKeyExchange)}, agentKeyExchangeMsgContentBytes...)
	if err := p.dataChannel.SendKeyExchangeMessage(agentKeyExchangeMsgBytes); err != nil {
		p.logger.WithError(err).Error("Unable to send agent key exchange message")
		return err
	}
	// p.logger.Infof("Sent AgentKeyExchange Message: %+v\n", agentKeyExchangeMsg)
	return nil
}

func (p *ShellPlugin) ECDHEClientKeyExchangeMessageHandler(payload []byte) error {
	// 收到ClientKeyExchange报文，继续密钥协商流程
	p.keyExchangeState = RecClientKeyExchange
	// 解析ECDHEClientKeyExchange报文
	var ecdheClientKeyExchangeMsg ECDHEClientKeyExchangeMsg
	if err := json.Unmarshal(payload[1:], &ecdheClientKeyExchangeMsg); err != nil {
		p.logger.WithError(err).Error("Invalid ECDHE client key exchange message content")
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage(fmt.Sprintf("Invalid ECDHE client key exchange message content: %+v", ecdheClientKeyExchangeMsg), "ClientKeyExchangeParseError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		return err
	}
	// p.logger.Infof("Received ECDHEClientKeyExchange Message: %+v\n", ecdheClientKeyExchangeMsg)
	// 当前AgentHello默认clientCertificate为false，ClientKeyExchange中不包含signature字段，因此未处理ecdheClientKeyExchangeMsg.signature字段
	// base64解码clientPub
	clientPublicKey, err := base64.StdEncoding.DecodeString(ecdheClientKeyExchangeMsg.ClientPub)
	if err != nil {
		p.logger.WithError(err).Error("Client public key base64 decode Error")
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage("Client public key base64 decode Error", "ClientPublicKeyDecodeError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		return err
	}
	// 生成Agent公钥
	agentPrivateKey, agentPublicKey, err := encryptionutil.GeneratePublicKey(p.logger)
	if err != nil {
		p.logger.WithError(err).Errorf("Failed to generate ECDHE public key")
		// 发送密钥协商错误报文
		if sendErr := p.sendKeyExchangeErrorMessage("Failed to generate ECDHE public key", "AgentKeyGenerateError"); sendErr != nil {
			p.logger.WithError(sendErr).Error("Failed to send key exchange error message")
		}
		return err
	}
	// ECDH生成对称密钥dataKey
	dataKey, err := encryptionutil.GenerateSharedSecret(agentPrivateKey, clientPublicKey)
	if err != nil {
		p.logger.WithError(err).Error("Failed to generate dataKey with ECDH")
	}
	p.dataKey = string(dataKey)
	// p.logger.Infof("dataKey(base64): %s", base64.StdEncoding.EncodeToString(dataKey))
	// 创建cbc对象用于AES加密
	if err := p.InitBlock([]byte(p.dataKey)); err != nil {
		p.logger.WithError(err).Errorf("Failed to init CBC, dataKey='%v'", p.dataKey)
		return err
	}
	// 创建ECDHEAgentKeyExchange报文
	ecdheAgentKeyExchangeMsg := &ECDHEAgentKeyExchangeMsg{
		AgentPub: base64.StdEncoding.EncodeToString(agentPublicKey),
	}
	ecdheAgentKeyExchangeMsgContentBytes, err := json.Marshal(ecdheAgentKeyExchangeMsg)
	if err != nil {
		p.logger.WithError(err).Errorf("Invalid ECDHEAgentKeyExchangeMsgBytes content")
		return err
	}
	// 添加actionType（content的第一个byte）
	ecdheAgentKeyExchangeMsgBytes := append([]byte{byte(AgentKeyExchange)}, ecdheAgentKeyExchangeMsgContentBytes...)
	if err := p.dataChannel.SendKeyExchangeMessage(ecdheAgentKeyExchangeMsgBytes); err != nil {
		p.logger.WithError(err).Errorf("Unable to send ECDHE agent key exchange message")
		return err
	}
	// p.logger.Infof("Sent ECDHEAgentKeyExchangeMsg Message: %+v\n", ecdheAgentKeyExchangeMsg)
	return nil
}

func (p *ShellPlugin) sendKeyExchangeErrorMessage(message string, code string) error {
	keyExchangeErrorMsp := &KeyExchangeErrorMsg{
		Code:    "KeyExchangeError." + code,
		Message: message,
	}
	keyExchangeErrorMsgBytes, err := json.Marshal(keyExchangeErrorMsp)
	if err != nil {
		return err
	}
	payload := append([]byte{byte(KeyExchangeError)}, keyExchangeErrorMsgBytes...)
	if err := p.dataChannel.SendKeyExchangeMessage(payload); err != nil {
		return err
	}
	return nil
}

func (p *ShellPlugin) InitBlock(key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	p.block = block
	return nil
}
