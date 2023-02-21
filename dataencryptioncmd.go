package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aliyun/aliyun_assist_client/agent/cryptdata"
	"github.com/aliyun/aliyun_assist_client/agent/ipc/client"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/cli"
	"github.com/aliyun/aliyun_assist_client/thirdparty/aliyun-cli/i18n"
	"github.com/sirupsen/logrus"
)

const (
	GenKeyPairFlagName = "genKeyPair"
	EncryptFlagName = "encrypt"
	DecryptFlagName = "decrypt"
	CheckKeyPairFlagName = "check"
	KeyPairIdFlagName = "keyPairId"
	KeyPairTimeoutFlagName = "timeout"
	DataFlagName = "text"

	DataEncryptSubCmd = "data-encryption"
)

var (
	dataEncryptionFlags = []cli.Flag{
		{
			Name: GenKeyPairFlagName,
			Shorthand: 'g',
			Short: i18n.T(
				`Generate a key pair with RSA-OAEP`, 
				`使用RSA-OAEP算法生成一个秘钥对`,
			),
			AssignedMode: cli.AssignedNone,
			Category: "caller",
		},
		{
			Name: EncryptFlagName,
			Shorthand: 'e',
			Short: i18n.T(
				`Use the key pair specified by keyPairId to encrypt text, and output the ciphertext encoded by base64`, 
				`使用 keyPairId 指定的密钥对加密文本数据，输出被使用base64编码后的密文`,
			),
			AssignedMode: cli.AssignedNone,
			Category: "caller",
		},
		{
			Name: DecryptFlagName,
			Shorthand: 'd',
			Short: i18n.T(
				`Decode the ciphertext with base64 and then decrypt it with the key pair specified by keyPairId`, 
				`将密文使用base64解码后再使用 keyPairId 指定的密钥对解密`,
			),
			AssignedMode: cli.AssignedNone,
			Category: "caller",
		},
		{
			Name: CheckKeyPairFlagName,
			Shorthand: 'c',
			Short: i18n.T(
				`Check key pair list or the public key of a specified key pair by keyPairId`, 
				`查看秘钥对列表或者通过 keyPairId 查看指定的公钥`,
			),
			AssignedMode: cli.AssignedNone,
			Category: "caller",
		},
		{
			Name: KeyPairIdFlagName,
			Shorthand: 'i',
			Short: i18n.T(
				`Id of key pair`, 
				`秘钥对的id`,
			),
			AssignedMode: cli.AssignedOnce,
			Category: "caller",
		},
		{
			Name: KeyPairTimeoutFlagName,
			Shorthand: 't',
			Short: i18n.T(
				`Key pair will expire after <timeout> seconds, default is 60s`, 
				`秘钥对的过期时间，默认60秒`,
			),
			AssignedMode: cli.AssignedOnce,
			Category: "caller",
		},
		{
			Name: DataFlagName,
			Shorthand: 'T',
			Short: i18n.T(
				`Text needed to be decrypted or encrypted, the max length of text to be encrypted is 190 bytes`,
				`需要被加密或者解密的文本内容，对于加密操作限制最大文本长度为190字节`),
			AssignedMode: cli.AssignedOnce,
			Category: "caller",
		},
		{
			Name: JsonFlagName,
			Short: i18n.T(
				`Output in JSON format`,
				`以JSON格式输出`),
			AssignedMode: cli.AssignedNone,
			Category: "caller",
		},
	}

	dataEncryptionCmd = cli.Command{
		Name: DataEncryptSubCmd,
		Short:             i18n.T(
								"Use the RSA-OAEP algorithm to encrypt or decrypt text, the public modulus of the secret key is 2048 bit, the hash function used is sha256, and the max length of text to be encrypted is 190 bytes", 
								"使用RSA-OAEP算法对文本内容进行加密或者解密，秘钥的公共模数为2048bit，使用的hash函数是sha256，可以加密的最长文本是190字节",
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
	encryptFlag := ctx.Flags().Get(EncryptFlagName).IsAssigned()
	decryptFlag := ctx.Flags().Get(DecryptFlagName).IsAssigned()
	checkKeyFlag := ctx.Flags().Get(CheckKeyPairFlagName).IsAssigned()
	keyPairId, _ := ctx.Flags().Get(KeyPairIdFlagName).GetValue()
	keyPairTimeout, _ := ctx.Flags().Get(KeyPairTimeoutFlagName).GetValue()
	data, _ := ctx.Flags().Get(DataFlagName).GetValue()
	jsonFlag := ctx.Flags().Get(JsonFlagName).IsAssigned()

	// Necessary initialization work
	log.InitLog("aliyun_assist_main.log", logPath)
	// Add field SubCmd to make log entries separated from the main process's
	log.GetLogger().SetFormatter(&log.CustomLogrusTextFormatter{
		Fileds: logrus.Fields{
			"SubCmd": DataEncryptSubCmd,
		},
	})
	timeout := 60
	if keyPairTimeout != "" {
		if t, err := strconv.Atoi(keyPairTimeout); err != nil || t <= 0 {
			fmt.Printf("Invalid param, `%s` needs to be a positive integer.\n", KeyPairTimeoutFlagName)
			cli.Exit(1)
		} else {
			timeout = t
		}
	}
	if keyPairId != "" && len(keyPairId) > 32 {
		fmt.Printf("Invalid param, length of `%s` needs less than or equal to 32.\n", KeyPairIdFlagName)
		cli.Exit(1)
	}
	if genKeyPairFlag {
		keyInfo, errCode, err := client.GenRsaKeyPair(keyPairId, timeout)
		if err != nil {
			fmt.Println("Generate key pair failed: ", err)
			cli.Exit(int(errCode))
		} else {
			if jsonFlag {
				if output, err := json.MarshalIndent(keyInfo, "", "\t"); err != nil {
					fmt.Println("Generate key pair failed: ", err)
				} else {
					fmt.Println(string(output))
				}
			} else {
				fmt.Printf("%s\n%s", keyInfo.Id, keyInfo.PublicKey)
			}
		}
	} else if encryptFlag {
		if keyPairId == "" || data == "" {
			fmt.Printf("Params `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName)
			cli.Exit(1)
		}
		byteData := []byte(data)
		if len(byteData) > cryptdata.LIMIT_PLAINTEXT_LEN {
			fmt.Printf("Max length of data to encrypt is %d bytes, but real length is %d", cryptdata.LIMIT_PLAINTEXT_LEN, len(byteData))
			cli.Exit(1)
		}
		cipherText, errCode, err := client.EncryptText(keyPairId, data)
		if err != nil {
			fmt.Printf("Encrypt data with key pair[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		} else {
			fmt.Print(cipherText)
		}
	} else if decryptFlag {
		if keyPairId == "" || data == "" {
			fmt.Printf("Params `%s` and `%s` can not be empty.\n", KeyPairIdFlagName, DataFlagName)
			cli.Exit(1)
		}
		plainText, errCode, err := client.DecryptText(keyPairId, data)
		if err != nil {
			fmt.Printf("Decrypt data with key pair[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		} else {
			fmt.Print(plainText)
		}
	} else if checkKeyFlag {
		output, errCode, err := client.CheckKey(keyPairId, jsonFlag)
		if err != nil {
			fmt.Printf("CheckKey with keyPairId[%s] failed: %s\n", keyPairId, err.Error())
			cli.Exit(int(errCode))
		} else {
			fmt.Println(output)
		}
	}
	return nil
}

func sample() string {
	s := ""
	s += fmt.Sprintf("aliyun-service %s --%s --%s abc --%s 120 (--%s)", DataEncryptSubCmd, GenKeyPairFlagName, KeyPairIdFlagName, KeyPairTimeoutFlagName, JsonFlagName)
	s += fmt.Sprintf("\n  aliyun-service %s --%s --%s abc --%s plain-text", DataEncryptSubCmd, EncryptFlagName, KeyPairIdFlagName, DataFlagName)
	s += fmt.Sprintf("\n  aliyun-service %s --%s --%s abc --%s cipher-text", DataEncryptSubCmd, DecryptFlagName, KeyPairIdFlagName, DataFlagName)
	s += fmt.Sprintf("\n  aliyun-service %s --%s --%s abc (--%s)", DataEncryptSubCmd, CheckKeyPairFlagName, KeyPairIdFlagName, JsonFlagName)
	s += fmt.Sprintf("\n  aliyun-service %s --%s (--%s)", DataEncryptSubCmd, CheckKeyPairFlagName, JsonFlagName)
	return s
}