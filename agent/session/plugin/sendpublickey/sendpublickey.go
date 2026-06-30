package sendpublickey

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/session"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"
)

const (
	SENDSSHPUBLICKEY_PUBLICCOMMANDID = "cmd-ACS-ECS-SendSshPublicKey-linux.sh"

	INSTALL_PLUGIN_COMMAND = "acs-plugin-manager -e -P config_ecs_instance_connect --params --install"
)

var installPluginCommandBase64 string = base64.StdEncoding.EncodeToString([]byte(INSTALL_PLUGIN_COMMAND))

func NewSendPublicKeyCommand() *cli.Command {

	c := &cli.Command{
		Name: "send_public_key",
		Short: i18n.T(
			"use send_public_key to send a temporary ssh public key to instance",
			"使用 send_public_key 向实例下发一个临时的ssh公钥"),
		Usage: "send_public_key --instance {instance_id}  --public_key {public_key}",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			instance_id, _ := config.InstanceFlag(ctx.Flags()).GetValue()
			public_key, _ := config.PublicKeyFlag(ctx.Flags()).GetValue()
			user_name, _ := config.UserNameFlag(ctx.Flags()).GetValue()
			if instance_id == "" || public_key == "" {
				fmt.Println("params `instance` and `public-key` are necessary")
				return nil
			}
			return doSession(ctx, instance_id, public_key, user_name)
		},
	}

	return c
}

func doSession(ctx *cli.Context, instance_id, public_key, user_name string) error {
	// 判断public_key 是公钥内容还是公钥文件路径
	isfile := false
	s, err := os.Stat(public_key)
	if err != nil {
		if os.IsExist(err) {
			isfile = !s.IsDir()
		}
	} else {
		isfile = !s.IsDir()
	}
	publicKeyContent := ""
	if isfile {
		publicKeyContent, err = readFile(public_key)
		if err != nil {
			fmt.Printf("read public_key file failed %s\n", err.Error())
			return err
		}
	} else {
		publicKeyContent = public_key
	}

	if user_name == "" {
		user_name = "root"
	}
	client, err := session.GetEcsClient(ctx)
	if err != nil {
		return err
	}
	// 执行agent命令下发，安装config_ecs_instance_connect插件
	runcommandRequest := ecsclient.RunCommandRequest{
		Type:            tea.String("RunShellScript"),
		CommandContent:  &installPluginCommandBase64,
		ContentEncoding: tea.String("Base64"),
		Timeout:         tea.Int64(5),
		RegionId:        client.RegionId,
		InstanceId:      []*string{tea.String(instance_id)},
	}

	runcommandResponse, err := client.RunCommand(&runcommandRequest)
	if err != nil {
		fmt.Printf("install config_ecs_instance_connect failed %s\n", err)
		return err
	}
	invokedId := runcommandResponse.Body.InvokeId
	time.Sleep(time.Duration(3) * time.Second)
	// 检查安装插件的命令是否执行成功
	describeInvocationRequest := ecsclient.DescribeInvocationResultsRequest{
		RegionId:   client.RegionId,
		InvokeId:   invokedId,
		InstanceId: tea.String(instance_id),
	}
	describeInvocationResponse, err := client.DescribeInvocationResults(&describeInvocationRequest)
	if err != nil {
		fmt.Printf("query 'install config_ecs_instance_connect' command result failed %s\n", err.Error())
		return err
	}
	invocationStatus := *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].InvocationStatus
	for invocationStatus == "Running" || invocationStatus == "Pending" {
		time.Sleep(time.Duration(3) * time.Second)
		describeInvocationResponse, err = client.DescribeInvocationResults(&describeInvocationRequest)
		if err != nil {
			fmt.Printf("query 'install config_ecs_instance_connect' command result failed %s\n", err.Error())
			return err
		}
		invocationStatus = *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].InvocationStatus
	}
	if invocationStatus != "Success" {
		fmt.Printf("'install config_ecs_instance_connect' command failed, InvocationStatus: %s\n", invocationStatus)
		return errors.New("'install config_ecs_instance_connect' command failed")
	}

	// 调用公共命令注册临时公钥
	invokecommandRequest := ecsclient.InvokeCommandRequest{
		CommandId:  tea.String(SENDSSHPUBLICKEY_PUBLICCOMMANDID),
		InstanceId: []*string{tea.String(instance_id)},
		RegionId:   client.RegionId,
		Parameters: map[string]interface{}{
			"username":     user_name,
			"sshPublicKey": publicKeyContent,
		},
	}
	invokecommandResponse, err := client.InvokeCommand(&invokecommandRequest)
	if err != nil {
		fmt.Printf("run public command ACS-ECS-SendSshPublicKey-linux failed %s\n", err.Error())
		return err
	}
	invokedId = invokecommandResponse.Body.InvokeId
	describeInvocationRequest = ecsclient.DescribeInvocationResultsRequest{
		RegionId:   client.RegionId,
		InvokeId:   invokedId,
		InstanceId: tea.String(instance_id),
	}

	describeInvocationResponse, err = client.DescribeInvocationResults(&describeInvocationRequest)
	if err != nil {
		fmt.Printf("query 'ACS-ECS-SendSshPublicKey-linux' command result failed %s\n", err.Error())
		return err
	}

	invocationStatus = *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].InvocationStatus
	for invocationStatus == "Running" || invocationStatus == "Pending" {
		time.Sleep(time.Duration(3) * time.Second)
		describeInvocationResponse, err = client.DescribeInvocationResults(&describeInvocationRequest)
		if err != nil {
			fmt.Printf("query 'ACS-ECS-SendSshPublicKey-linux' command result failed %s\n", err.Error())
			return err
		}
		invocationStatus = *describeInvocationResponse.Body.Invocation.InvocationResults.InvocationResult[0].InvocationStatus
	}
	if invocationStatus != "Success" {
		fmt.Printf("'ACS-ECS-SendSshPublicKey-linux' command failed, InvocationStatus: %s\n", invocationStatus)
		return errors.New("'ACS-ECS-SendSshPublicKey-linux' command failed")
	}

	fmt.Println("The temporary ssh_public_key has been registered")
	return nil
}

func readFile(filename string) (content string, err error) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return
	}
	content = string(b)
	return
}
