package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"

	"github.com/aliyun/aliyun_assist_client/agent/cryptdata"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/interprocess/cryptdata/client"
)

const (
	TimeoutFlagName         = "timeout"
	JsonFlagName            = "json"
	GenKeyPairFlagName      = "create-keypair"
	RmKeyPairFlagName       = "remove-keypair"
	CreateSecretParam       = "create-secret"
	GetSecretParamValue     = "get-secret"
	EncryptFlagName         = "encrypt"
	DecryptFlagName         = "decrypt"
	CheckKeyPairFlagName    = "get-keypair"
	SignDataFlagName        = "sign"
	VerifySignatureFlagName = "verify-sign"
	KeyPairIdFlagName       = "keypair-id"
	KeyPairTimeoutFlagName  = "timeout"
	DataFlagName            = "text"
	SecretName              = "param-name"
	AesSecuredFlagName      = "aes-secured"
	SignatureFlagName       = "signature"

	DataEncryptSubCmd = "data-encryption"

	// These aliases are provided to maintain compatibility with older version
	// parameter names, the new parameters should not be set with aliases.
	GenKeyPairFlagAliase      = "genKeyPair"
	RmKeyPairFlagAliase       = "rmKeyPair"
	CreateSecretParamAliase   = "createSecret"
	GetSecretParamValueAliase = "getSecretValue"
	KeyPairIdFlagAliase       = "keyPairId"
	SecretNameAliase          = "paramName"
)

var (
	dataEncryptionFlags = []cli.Flag{
		{
			Name:      GenKeyPairFlagName,
			Aliases:   []string{GenKeyPairFlagAliase},
			Shorthand: 'g',
			Short: i18n.T(
				`Generate a keypair with RSA-OAEP.`,
				`使用RSA-OAEP算法生成一个秘钥对。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:      RmKeyPairFlagName,
			Aliases:   []string{RmKeyPairFlagAliase},
			Shorthand: 'r',
			Short: i18n.T(
				`Remove a keypair by keypair-id.`,
				`删除指定的秘钥对。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:      CreateSecretParam,
			Aliases:   []string{CreateSecretParamAliase},
			Shorthand: 's',
			Short: i18n.T(
				`Create a secret param.`,
				`创建加密参数。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:    GetSecretParamValue,
			Aliases: []string{GetSecretParamValueAliase},
			Short: i18n.T(
				`Get secret param's value.`,
				`获取加密参数的值。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:      EncryptFlagName,
			Shorthand: 'e',
			Short: i18n.T(
				`Use the keypair specified by keypair-id to encrypt text, and output the ciphertext encoded by base64.`,
				`使用 keypair-id 指定的密钥对加密文本数据，输出被使用base64编码后的密文。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:      DecryptFlagName,
			Shorthand: 'd',
			Short: i18n.T(
				`Decode the ciphertext with base64 and then decrypt it with the keypair specified by keypair-id.`,
				`将密文使用base64解码后再使用 keypair-id 指定的密钥对解密。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name:      CheckKeyPairFlagName,
			Shorthand: 'c',
			Short: i18n.T(
				`Get keypair list or the public key of a specified keypair by keypair-id.`,
				`查看秘钥对列表或者通过 keypair-id 查看指定的公钥。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name: SignDataFlagName,
			Short: i18n.T(

				`Sign the data using the private key of a specified keypair by keypair-id.
                    	Use PSS signature scheme and sha256 hash algorithm.`,
				`使用 keypair-id 指定密钥对的私钥对数据进行签名,使用PSS签名方案和sha256哈希算法。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
		{
			Name: VerifySignatureFlagName,
			Short: i18n.T(
				`Verify the signature of the data using the public key of a specified keypair by keypair-id.
                    	Use PSS signature scheme and sha256 hash algorithm.`,
				`使用 keypair-id 指定密钥对的公钥验证数据的签名,使用PSS签名方案和sha256哈希算法。`,
			),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},

		{
			Name:      KeyPairIdFlagName,
			Aliases:   []string{KeyPairIdFlagAliase},
			Shorthand: 'i',
			Short: i18n.T(
				`Id of keypair.`,
				`秘钥对的id。`,
			),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:      KeyPairTimeoutFlagName,
			Shorthand: 't',
			Short: i18n.T(
				`Keypair or secret param will expire after <timeout> seconds, default is 60s.`,
				`秘钥对或者加密参数的过期时间，默认60秒。`,
			),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:      DataFlagName,
			Shorthand: 'T',
			Short: i18n.T(
				`Text needed to be decrypted or encrypted, the max length of text to be encrypted is 190 bytes.`,
				`需要被加密或者解密的文本内容，对于加密操作限制最大文本长度为190字节。`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name:      SecretName,
			Aliases:   []string{SecretNameAliase},
			Shorthand: 'n',
			Short: i18n.T(
				`Name of secret param.`,
				`加密参数的名称。`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name: AesSecuredFlagName,
			Short: i18n.T(
				`AES key which be encrypted (Base64 encoded).`,
				`加密后的AES秘钥 (Base64编码)。`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name: SignatureFlagName,
			Short: i18n.T(
				`Signature (Base64 encoded).`,
				`Base64编码后的签名。`),
			AssignedMode: cli.AssignedOnce,
			Category:     "caller",
		},
		{
			Name: JsonFlagName,
			Short: i18n.T(
				`Output in JSON format`,
				`以JSON格式输出。`),
			AssignedMode: cli.AssignedNone,
			Category:     "caller",
		},
	}

	dataEncryptionCmd = &cli.Command{
		Name: DataEncryptSubCmd,
		Short: i18n.T(
			"Use the RSA-OAEP algorithm to encrypt or decrypt text, the public modulus of the secret key is 2048 bit, the hash function used is sha256, and the max length of text to be encrypted is 190 bytes.",
			"使用RSA-OAEP算法对文本内容进行加密或者解密，秘钥的公共模数为2048bit，使用的hash函数是sha256，可以加密的最长文本是190字节。",
		),
		Usage:             fmt.Sprint(DataEncryptSubCmd, " [flags]"),
		Sample:            sample(),
		EnableUnknownFlag: false,
		Run:               runDataEncryptionCmd,
	}
)

func init() {
	for j := range dataEncryptionFlags {
		dataEncryptionCmd.Flags().Add(&dataEncryptionFlags[j])
	}
}

func runDataEncryptionCmd(ctx *cli.Context, args []string) error {
	// Extract value of persistent flags
	logPath, _ := ctx.Flags().Get(LogPathFlagName).GetValue()
	// Extract value of flags just for the command
	genKeyPairFlag := ctx.Flags().Get(GenKeyPairFlagName).IsAssigned()
	rmKeyPairFlag := ctx.Flags().Get(RmKeyPairFlagName).IsAssigned()
	createSecret := ctx.Flags().Get(CreateSecretParam).IsAssigned()
	getSecretValue := ctx.Flags().Get(GetSecretParamValue).IsAssigned()
	encryptFlag := ctx.Flags().Get(EncryptFlagName).IsAssigned()
	decryptFlag := ctx.Flags().Get(DecryptFlagName).IsAssigned()
	checkKeyFlag := ctx.Flags().Get(CheckKeyPairFlagName).IsAssigned()
	signDataFlag := ctx.Flags().Get(SignDataFlagName).IsAssigned()
	verifySignFlag := ctx.Flags().Get(VerifySignatureFlagName).IsAssigned()

	keyPairId, _ := ctx.Flags().Get(KeyPairIdFlagName).GetValue()
	keyPairTimeout, _ := ctx.Flags().Get(KeyPairTimeoutFlagName).GetValue()
	data, _ := ctx.Flags().Get(DataFlagName).GetValue()
	secretName, _ := ctx.Flags().Get(SecretName).GetValue()
	aesKey, _ := ctx.Flags().Get(AesSecuredFlagName).GetValue()
	signature, _ := ctx.Flags().Get(SignatureFlagName).GetValue()
	jsonFlag := ctx.Flags().Get(JsonFlagName).IsAssigned()

	// Necessary initialization work
	log.InitLog("aliyun_assist_main.log", logPath, true)
	// IF write log failed, do nothing
	log.GetLogger().SetErrorCallback(func(error) {})
	// Add field SubCmd to make log entries separated from the main process's
	commonFields := log.DefaultCommonFields()
	commonFields["SubCmd"] = DataEncryptSubCmd
	log.GetLogger().SetFormatter(&log.CustomLogrusTextFormatter{
		CommonFields: commonFields,
	})

	timeout := 60
	if keyPairTimeout != "" {
		if t, err := strconv.Atoi(keyPairTimeout); err != nil || t <= 0 {
			fmt.Fprintf(os.Stderr, "Invalid param, `%s` needs to be a positive integer.\n", KeyPairTimeoutFlagName)
			cli.Exit(1)
		} else {
			timeout = t
		}
	}
	if keyPairId != "" && len(keyPairId) > 32 {
		fmt.Fprintf(os.Stderr, "Invalid param, length of `%s` needs less than or equal to 32.\n", KeyPairIdFlagName)
		cli.Exit(1)
	}
	if secretName != "" && len(secretName) > 32 {
		fmt.Fprintf(os.Stderr, "Invalid param, length of `%s` needs less than or equal to 32.\n", secretName)
		cli.Exit(1)
	}
	if genKeyPairFlag {
		keyInfo, errCode, err := client.GenRsaKeyPair(keyPairId, timeout)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Generate key pair failed: ", err)
			cli.Exit(int(errCode))
		} else {
			if jsonFlag {
				if output, err := json.MarshalIndent(keyInfo, "", "\t"); err != nil {
					fmt.Fprintln(os.Stderr, "Generate key pair failed: ", err)
					cli.Exit(1)
				} else {
					fmt.Println(string(output))
				}
			} else {
				fmt.Printf("%s\n%s", keyInfo.Id, keyInfo.PublicKey)
			}
		}
	} else if rmKeyPairFlag {
		if keyPairId == "" {
			fmt.Fprintf(os.Stderr, "Params `%s` can not be empty.\n", KeyPairIdFlagName)
			cli.Exit(1)
		}
		errCode, err := client.RmRsaKeyPair(keyPairId)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Remove key pair failed: ", err)
			cli.Exit(int(errCode))
		} else {
			fmt.Printf("Key pair [%s] has been removed\n", keyPairId)
		}
	} else if createSecret {
		if keyPairId == "" || data == "" || secretName == "" {
			fmt.Fprintf(os.Stderr, "Params `%s` and `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName, SecretName)
			cli.Exit(1)
		}
		paramInfo, errCode, err := client.CreateSecretParam(keyPairId, secretName, data, aesKey, int64(timeout))
		if err != nil {
			fmt.Fprintln(os.Stderr, "Create secret param failed: ", err)
			cli.Exit(int(errCode))
		} else {
			if output, err := json.MarshalIndent(paramInfo, "", "\t"); err != nil {
				fmt.Fprintln(os.Stderr, "Create secret param failed: ", err)
				cli.Exit(1)
			} else {
				fmt.Println(string(output))
			}
		}
	} else if getSecretValue {
		if secretName == "" {
			fmt.Fprintf(os.Stderr, "Params `%s` can not be empty.\n", SecretName)
			cli.Exit(1)
		}
		paramValueInfo, errCode, err := client.GetSecretParamValue(secretName)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Get secret param value failed: ", err)
			cli.Exit(int(errCode))
		}
		if jsonFlag {
			if output, err := json.MarshalIndent(paramValueInfo, "", "\t"); err != nil {
				fmt.Fprintln(os.Stderr, "Get secret param value failed: ", err)
				cli.Exit(1)
			} else {
				fmt.Println(string(output))
			}
		} else {
			fmt.Println(paramValueInfo.SecretValue)
		}
	} else if encryptFlag {
		if keyPairId == "" || data == "" {
			fmt.Fprintf(os.Stderr, "Params `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName)
			cli.Exit(1)
		}
		byteData := []byte(data)
		if len(byteData) > cryptdata.LIMIT_PLAINTEXT_LEN {
			fmt.Fprintf(os.Stderr, "Max length of data to encrypt is %d bytes, but real length is %d", cryptdata.LIMIT_PLAINTEXT_LEN, len(byteData))
			cli.Exit(1)
		}
		cipherText, errCode, err := client.EncryptText(keyPairId, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Encrypt data with key pair[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		} else {
			fmt.Print(cipherText)
		}
	} else if decryptFlag {
		if keyPairId == "" || data == "" {
			fmt.Fprintf(os.Stderr, "Params `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName)
			cli.Exit(1)
		}
		plainText, errCode, err := client.DecryptText(keyPairId, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Decrypt data with key pair[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		} else {
			fmt.Print(plainText)
		}
	} else if checkKeyFlag {
		output, errCode, err := client.CheckKey(keyPairId, jsonFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "CheckKey with keyPairId[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		} else {
			fmt.Println(output)
		}
	} else if signDataFlag {
		if keyPairId == "" || data == "" {
			fmt.Fprintf(os.Stderr, "Params `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName)
			cli.Exit(1)
		}
		signature, errCode, err := client.SignData(keyPairId, data)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Sign data with key pair[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		}
		fmt.Println(signature)
	} else if verifySignFlag {
		if keyPairId == "" || data == "" || signature == "" {
			fmt.Fprintf(os.Stderr, "Params `%s`, `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName, SignatureFlagName)
			cli.Exit(1)
		}

		valid, errCode, err := client.VerifySignature(keyPairId, data, signature)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Verify signature with key pair[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		}
		if valid {
			fmt.Println("valid")
			cli.Exit(0)
		}
		fmt.Println("invalid")
		cli.Exit(1)

	}
	return nil
}

func sample() string {
	s := ""
	path, _ := os.Executable()
	_, exeFile := filepath.Split(path)
	s += fmt.Sprintf("%s %s --%s --%s <keypair-id> --%s <timeout> [--%s]", exeFile, DataEncryptSubCmd, GenKeyPairFlagName, KeyPairIdFlagName, KeyPairTimeoutFlagName, JsonFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> [--%s]", exeFile, DataEncryptSubCmd, CheckKeyPairFlagName, KeyPairIdFlagName, JsonFlagName)
	s += fmt.Sprintf("\n  %s %s --%s [--%s]", exeFile, DataEncryptSubCmd, CheckKeyPairFlagName, JsonFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id>", exeFile, DataEncryptSubCmd, RmKeyPairFlagName, KeyPairIdFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> --%s <plain-text>", exeFile, DataEncryptSubCmd, EncryptFlagName, KeyPairIdFlagName, DataFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> --%s <cipher-text>", exeFile, DataEncryptSubCmd, DecryptFlagName, KeyPairIdFlagName, DataFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> --%s <cipher-text> --%s <param-name> --%s <timeout>", exeFile, DataEncryptSubCmd, CreateSecretParam, KeyPairIdFlagName, DataFlagName, SecretName, TimeoutFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> --%s <cipher-text> --%s <param-name> --%s <encrypted-aes-key> --%s <timeout>", exeFile, DataEncryptSubCmd, CreateSecretParam, KeyPairIdFlagName, DataFlagName, SecretName, AesSecuredFlagName, TimeoutFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <param-name> [--%s]", exeFile, DataEncryptSubCmd, GetSecretParamValue, SecretName, JsonFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> --%s <plain-text>", exeFile, DataEncryptSubCmd, SignDataFlagName, KeyPairIdFlagName, DataFlagName)
	s += fmt.Sprintf("\n  %s %s --%s --%s <keypair-id> --%s <plain-text> --%s <signature>", exeFile, DataEncryptSubCmd, VerifySignatureFlagName, KeyPairIdFlagName, DataFlagName, SignatureFlagName)

	return s
}
