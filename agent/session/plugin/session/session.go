// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package session

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/client"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/constant"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/log"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
)

const (
	GenKey_CmdID_Linux   = "ACS-ECS-GenerateRsaKeypair-linux.sh"
	GenKey_CmdID_Windows = "ACS-ECS-GenerateRsaKeypair-windows.ps1"

	CreateSecret_CmdID_Linux   = "ACS-ECS-CreateSecret-for-linux.sh"
	CreateSecret_CmdID_Windows = "ACS-ECS-CreateSecret-for-windows.ps1"
)

func NewSessionCommand() *cli.Command {

	c := &cli.Command{
		Name: "session",
		Short: i18n.T(
			"use session manager devops aliyun ecs instance",
			"使用session manager运维阿里云实例"),
		Usage: "session --instance {instance_id} [--user-name {user_name}] [--idle-timeout {idle_timeout}]",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			instance_id, _ := config.InstanceFlag(ctx.Flags()).GetValue()
			wss_url, _ := config.WssUrlFlag(ctx.Flags()).GetValue()
			user_name, _ := config.UserNameFlag(ctx.Flags()).GetValue()
			idle_timeout, _ := config.IdleTimeoutFlag(ctx.Flags()).GetValue()
			connection_type, _ := config.ConnectionTypeFlag(ctx.Flags()).GetValue()
			passwd_name, _ := config.PasswdNameFlag(ctx.Flags()).GetValue()
			passwd_content, _ := config.PasswdFlag(ctx.Flags()).GetValue()
			var idleTimeout int64
			var err error
			if idle_timeout == "" {
				// Default value is 180, because Agent will disconnect if no
				// package received within 180 seconds.
				idleTimeout = 180
			} else {
				idleTimeout, err = strconv.ParseInt(idle_timeout, 10, 32)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Parse param `%s` failed: %v", config.IdleTimeoutFlagName, err)
					os.Exit(1)
				}
			}
			if connection_type == "" {
				connection_type = constant.CONNECTION_TYPE_INTERNET
			} else if connection_type != constant.CONNECTION_TYPE_INTERNET && connection_type != constant.CONNECTION_TYPE_INTRANET {
				fmt.Fprintf(os.Stderr, "Invalid param `%s`\n", config.ConnectionTypeFlagName)
				os.Exit(1)
			}
			if passwd_name != "" && passwd_content != "" {
				fmt.Fprintf(os.Stderr, "Can not specify both `%s` and `%s` parameters at the same time", config.PasswdNameFlagName, config.PasswdFlagName)
				os.Exit(1)
			}

			return doSession(ctx, instance_id, user_name, passwd_name, passwd_content, wss_url, int32(idleTimeout), connection_type)
		},
	}

	return c
}

type Response struct {
	//	Code            string
	AccessKeyId     string
	AccessKeySecret string
	SecurityToken   string
	Expiration      string
}

func getClientByCredentialsURI(credentialsURI string) (Response, error) {
	var response Response
	res, err := http.Get(credentialsURI)
	if err != nil {
		return response, fmt.Errorf("Get Credentials from %s failed", credentialsURI)
	}

	if res.StatusCode != 200 {
		return response, fmt.Errorf("Get Credentials from %s failed, status code %d", credentialsURI, res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return response, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return response, fmt.Errorf("Unmarshal credentials failed, the body %s", string(body))
	}

	//	if response.Code != "Success" {
	//		return response, fmt.Errorf("Get sts token err, Code is not Success")
	//	}

	return response, nil
}

func GetEcsClient(ctx *cli.Context) (*ecsclient.Client, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		fmt.Printf("load configuration failed %s\n", err)
		return nil, fmt.Errorf("load configuration failed %s", err)
	}

	clientConf := openapi.Config{}

	if profile.Mode == "AK" {
		clientConf.RegionId = tea.String(profile.RegionId)
		clientConf.AccessKeyId = tea.String(profile.AccessKeyId)
		clientConf.AccessKeySecret = tea.String(profile.AccessKeySecret)

	} else if profile.Mode == "StsToken" {
		clientConf.RegionId = tea.String(profile.RegionId)
		clientConf.AccessKeyId = tea.String(profile.AccessKeyId)
		clientConf.AccessKeySecret = tea.String(profile.AccessKeySecret)
		clientConf.SecurityToken = tea.String(profile.StsToken)
	} else if profile.Mode == "CredentialsURI" {
		res, err := getClientByCredentialsURI(profile.CredentialsURI)
		if err != nil {
			return nil, err
		}
		log.GetLogger().Infof("CredentialsURI: %s, %s, %s,%s", profile.RegionId, res.AccessKeyId, res.AccessKeySecret, res.SecurityToken)
		clientConf.RegionId = tea.String(profile.RegionId)
		clientConf.AccessKeyId = tea.String(res.AccessKeyId)
		clientConf.AccessKeySecret = tea.String(res.AccessKeySecret)
		clientConf.SecurityToken = tea.String(res.SecurityToken)
	} else {
		fmt.Printf("load configuration failed")
		return nil, fmt.Errorf("cound not support current auth mode")
	}

	return ecsclient.NewClient(&clientConf)
}

func GetComputeNestSupplierClient(ctx *cli.Context) (*sdk.Client, string, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		fmt.Errorf("load configuration failed %s", err)
		return nil, "", fmt.Errorf("load configuration failed %s", err)
	}

	client, err := profile.GetClient(ctx)
	return client, profile.RegionId, err

}

func CheckSessionEnabled(ctx *cli.Context) {
	path, _ := os.Executable()
	config_dir, _ := filepath.Abs(filepath.Dir(path))

	_, err := os.Stat(config_dir + "/debugmode")
	if err == nil {
		return
	}

	client, err := GetEcsClient(ctx)
	if err != nil {
		log.GetLogger().Errorln(err)
		fmt.Print(err.Error())
		os.Exit(1)
	}

	request := ecsclient.DescribeUserBusinessBehaviorRequest{
		RegionId:  client.RegionId,
		StatusKey: tea.String("sessionManagerStatus"),
	}

	response, err := client.DescribeUserBusinessBehavior(&request)
	if err != nil {
		log.GetLogger().Errorln(err)
		fmt.Print(err.Error())
		os.Exit(1)
	}
	if *response.Body.StatusValue == "disabled" {
		log.GetLogger().Errorln("session manager is disabled, please enable first")
		fmt.Println("session manager is disabled, please enable first")
		os.Exit(1)
	}

}

func doSession(ctx *cli.Context, instance_id, user_name, passwd_name, passwd_content, wss_url string, idleTimeout int32, connectionType string) error {
	CheckSessionEnabled(ctx)
	var websocket_url string
	if instance_id != "" {
		client, err := GetEcsClient(ctx)
		if err != nil {
			fmt.Print(err.Error())
			log.GetLogger().Errorln(err)
			return fmt.Errorf("get ecs client err:%v", err)
		}

		if passwd_content != "" {
			passwd_name = fmt.Sprintf("session_%d", time.Now().UnixNano())
			osType := describeInstanceOsType(log.GetLogger(), client, instance_id)
			if err := createPasswdParam(log.GetLogger(), client, instance_id, osType, passwd_content, passwd_name); err != nil {
				fmt.Print(err.Error())
				log.GetLogger().Errorln(err)
				return fmt.Errorf("create passwd param failed:%v", err)
			}
		}

		request := ecsclient.StartTerminalSessionRequest{
			RegionId:       client.RegionId,
			InstanceId:     []*string{tea.String(instance_id)},
			Username:       &user_name,
			ConnectionType: &connectionType,
			PasswordName:   &passwd_name,
		}

		response, err := client.StartTerminalSession(&request)
		if err != nil {
			log.GetLogger().Errorln(err, response)
			fmt.Print(err.Error())
		}
		log.GetLogger().Infof("response is %#v\n", response)
		websocket_url = *response.Body.WebSocketUrl
	} else {
		websocket_url = wss_url
	}
	log.GetLogger().Infoln("wss url:", websocket_url)

	url := websocket_url
	url = strings.Replace(url, "sessionid", "sessionId", 1)
	log.GetLogger().Infoln("websocket url:", url)
	client, err := client.NewClient(url, os.Stdin, os.Stdout, false, "", false, config.VerboseFlag(ctx.Flags()).IsAssigned(), idleTimeout)
	if err != nil {
		log.GetLogger().Fatalf("Create client error: %v", err)
	}
	// loop
	go func() {
		waitSignals()
	}()
	if err = client.Loop(); err != nil {
		log.GetLogger().Fatalf("Communication error: %v", err)
	}

	return nil
}
func waitSignals() error {
	sigChan := make(chan os.Signal, 2)
	signal.Notify(
		sigChan,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	<-sigChan
	log.GetLogger().Infoln("session plugin stop", sigChan)
	os.Exit(1)

	return nil
}

type publicKey struct {
	ID string `json:"id"`
	CT int64  `json:"createdTimestamp"`
	ET int64  `json:"expiredTimestamp"`
	PK string `json:"publicKey"`
}

func createPasswdParam(logger logrus.FieldLogger, client *ecsclient.Client, instanceId, osType, content, paramName string) error {
	var genKeyCmd, createSecretCmd string
	switch osType {
	case "linux":
		genKeyCmd = GenKey_CmdID_Linux
		createSecretCmd = CreateSecret_CmdID_Linux
	case "windows":
		genKeyCmd = GenKey_CmdID_Windows
		createSecretCmd = CreateSecret_CmdID_Windows
	default:
		return fmt.Errorf("unsupported os type %s", osType)
	}
	output, exitCode, err := invokePublicCommand(logger, client, instanceId, genKeyCmd, nil)
	if err != nil {
		logger.WithError(err).Error("failed to generate key")
		return err
	}
	if exitCode != 0 {
		logger.WithField("exitCode", exitCode).Error("failed to generate key: ", output)
		return fmt.Errorf("failed to generate key: ", output)
	}
	pk := &publicKey{}
	err = json.Unmarshal([]byte(output), pk)
	if err != nil {
		logger.WithError(err).Error("unmarshal public key failed: ", output)
		return err
	}
	cipherTxt, err := encryptData(content, pk.PK)
	if err != nil {
		logger.WithError(err).Error("encrypt failed")
		return err
	}
	params := map[string]interface{}{
		"key_pair_id": pk.ID,
		"secret_name": paramName,
		"secret_data": cipherTxt,
	}
	output, exitCode, err = invokePublicCommand(logger, client, instanceId, createSecretCmd, params)
	if err != nil {
		logger.WithError(err).Error("failed to create secret")
		return err
	}
	if exitCode != 0 {
		logger.WithField("exitCode", exitCode).Error("failed to create secret: ", output)
		return fmt.Errorf("failed to create secret: %s", output)
	}
	return nil
}

func describeInstanceOsType(logger logrus.FieldLogger, client *ecsclient.Client, instanceId string) (osType string) {
	request := ecsclient.DescribeInstancesRequest{
		RegionId:    client.RegionId,
		InstanceIds: tea.String(fmt.Sprintf("[\"%s\"]", instanceId)),
	}
	response, err := client.DescribeInstances(&request)
	if err != nil {
		logger.WithError(err).Errorln("DescribeInstances failed")
		fmt.Print(err.Error())
		os.Exit(1)
	}
	if len(response.Body.Instances.Instance) != 1 {
		logger.Errorln("DescribeInstances failed, invalid instance count")
		fmt.Println("DescribeInstances failed, invalid instance count")
		os.Exit(1)
	}
	osType = *response.Body.Instances.Instance[0].OSType
	return
}

func invokePublicCommand(logger logrus.FieldLogger, client *ecsclient.Client, instanceId string, commandId string, params map[string]interface{}) (string, int64, error) {
	invokecommandRequest := ecsclient.InvokeCommandRequest{
		CommandId:  &commandId,
		InstanceId: []*string{&instanceId},
		RegionId:   client.RegionId,
		Parameters: params,
	}
	invokecommandResponse, err := client.InvokeCommand(&invokecommandRequest)
	if err != nil {
		fmt.Printf("invoke command '%s' failed %s\n", commandId, err.Error())
		return "", 0, err
	}

	time.Sleep(time.Second)
	invokedId := invokecommandResponse.Body.InvokeId
	describeInvocationRequest := ecsclient.DescribeInvocationResultsRequest{
		RegionId:        client.RegionId,
		InvokeId:        invokedId,
		InstanceId:      &instanceId,
		ContentEncoding: tea.String("PlainText"),
	}

	describeInvocationResponse, err := client.DescribeInvocationResults(&describeInvocationRequest)
	if err != nil {
		fmt.Printf("query command '%s' result failed %s\n", commandId, err.Error())
		return "", 0, err
	}

	if count := len(describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult); count != 1 {
		err = fmt.Errorf("invalid invocation result count[%d]", count)
		fmt.Printf("query command '%s' result failed %s\n", commandId, err.Error())
		return "", 0, err
	}
	invocationStatus := *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].InvocationStatus
	for invocationStatus == "Running" || invocationStatus == "Pending" {
		time.Sleep(time.Duration(2) * time.Second)
		describeInvocationResponse, err = client.DescribeInvocationResults(&describeInvocationRequest)
		if err != nil {
			fmt.Printf("query command '%s' result failed %s\n", commandId, err.Error())
			return "", 0, err
		}
		if count := len(describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult); count != 1 {
			err = fmt.Errorf("invalid invocation result count[%d]", count)
			fmt.Printf("query command '%s' result failed %s\n", commandId, err.Error())
			return "", 0, err
		}
		invocationStatus = *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].InvocationStatus
	}
	var output string
	var exitCode int64 = -1
	if describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].Output != nil {
		output = *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].Output	
	}
	if describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].ExitCode != nil {
		exitCode = *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].ExitCode	
	}
	if invocationStatus != "Success" {
		fmt.Printf("invoke command '%s' failed, InvocationStatus[%s], output[%s], exitCode[%d]\n", commandId, invocationStatus, output, exitCode)
		return output, exitCode, fmt.Errorf("'%s' command failed", commandId)
	}
	return output, exitCode, nil
}

func encryptData(msg string, publicKeyStr string) (string, error) {
	block, _ := pem.Decode([]byte(publicKeyStr))
	if block == nil {
		return "", fmt.Errorf("failed to parse PEM block containing the public key")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return "", err
	}
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("not an RSA public key")
	}

	cipher, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPubKey, []byte(msg), nil)
	if err != nil {
		return "", err
	}

	encryptedText := base64.StdEncoding.EncodeToString(cipher)
	return encryptedText, nil
}
