package sendpublickey

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	// "github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/session"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
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
		publicKeyContent, err = osutil.ReadFile(public_key)
		if err != nil {
			fmt.Errorf("read public_key file failed %s", err.Error())
			return err
		}
	} else {
		publicKeyContent = public_key
	}

	if user_name == "" {
		user_name = "root"
	}
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		fmt.Errorf("load configuration failed %s", err)
		return err
	}
	client, err := session.GetEcsClient(ctx)
	if err != nil {
		return err
	}
	// 执行agent命令下发，安装config_ecs_instance_connect插件
	runcommandRequest := ecs.CreateRunCommandRequest()
	runcommandRequest.Scheme = "https"
	runcommandRequest.Type = "RunShellScript"
	runcommandRequest.CommandContent = installPluginCommandBase64
	runcommandRequest.ContentEncoding = "Base64"
	runcommandRequest.Timeout = "5"
	runcommandRequest.RegionId = profile.RegionId
	runcommandRequest.InstanceId = &[]string{instance_id}
	runcommandResponse, err := client.RunCommand(runcommandRequest)
	if err != nil {
		fmt.Errorf("install config_ecs_instance_connect failed %s", err)
		return err
	}
	invokedId := runcommandResponse.InvokeId
	time.Sleep(time.Duration(3) * time.Second)
	// 检查安装插件的命令是否执行成功
	describeInvocationRequest := ecs.CreateDescribeInvocationResultsRequest()
	describeInvocationRequest.Scheme = "https"
	describeInvocationRequest.RegionId = profile.RegionId
	describeInvocationRequest.InvokeId = invokedId
	describeInvocationRequest.InstanceId = instance_id
	describeInvocationResponse, err := client.DescribeInvocationResults(describeInvocationRequest)
	if err != nil {
		fmt.Errorf("query 'install config_ecs_instance_connect' command result failed %s", err.Error())
		return err
	}
	for ; describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus == "Running" || describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus == "Pending"; {
		describeInvocationResponse, err = client.DescribeInvocationResults(describeInvocationRequest)
		if err != nil {
			fmt.Errorf("query 'install config_ecs_instance_connect' command result failed %s", err.Error())
			return err
		}
		time.Sleep(time.Duration(3) * time.Second)
	}
	if describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus != "Success" {
		fmt.Errorf("'install config_ecs_instance_connect' command failed, InvocationStatus: %s", describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus)
		return errors.New("'install config_ecs_instance_connect' command failed")
	}
	
	// 调用公共命令注册临时公钥
	invokecommandRequest := ecs.CreateInvokeCommandRequest()
	invokecommandRequest.Scheme = "https"
	invokecommandRequest.CommandId = SENDSSHPUBLICKEY_PUBLICCOMMANDID
	invokecommandRequest.InstanceId = &[]string{instance_id}
	invokecommandRequest.Parameters = map[string]interface{}{
		"username": user_name,
		"sshPublicKey": publicKeyContent,
	}
	invokecommandResponse, err := client.InvokeCommand(invokecommandRequest)
	if err != nil {
		fmt.Errorf("run public command ACS-ECS-SendSshPublicKey-linux failed %s", err.Error())
		return err
	}
	invokedId = invokecommandResponse.InvokeId
	describeInvocationRequest = ecs.CreateDescribeInvocationResultsRequest()
	describeInvocationRequest.Scheme = "https"
	describeInvocationRequest.RegionId = profile.RegionId
	describeInvocationRequest.InvokeId = invokedId
	describeInvocationResponse, err = client.DescribeInvocationResults(describeInvocationRequest)
	if err != nil {
		fmt.Errorf("query 'ACS-ECS-SendSshPublicKey-linux' command result failed %s", err.Error())
		return err
	}
	for ; describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus == "Running" || describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus == "Pending"; {
		describeInvocationResponse, err = client.DescribeInvocationResults(describeInvocationRequest)
		if err != nil {
			fmt.Errorf("query 'ACS-ECS-SendSshPublicKey-linux' command result failed %s", err.Error())
			return err
		}
		time.Sleep(time.Duration(2) * time.Second)
	}
	if describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus != "Success" {
		fmt.Errorf("'ACS-ECS-SendSshPublicKey-linux' command failed, InvocationStatus: %s", describeInvocationResponse.Invocation.InvocationResults.InvocationResult[0].InvocationStatus)
		return errors.New("'ACS-ECS-SendSshPublicKey-linux' command failed")
	}
	fmt.Println("The temporary ssh_public_key has been registered")
	return nil
}
