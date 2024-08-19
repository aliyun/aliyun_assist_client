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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/client"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/log"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
)

func NewSessionCommand() *cli.Command {

	c := &cli.Command{
		Name: "session",
		Short: i18n.T(
			"use session manager devops aliyun ecs instance",
			"使用session manager运维阿里云实例"),
		Usage: "session --instance {instance_id} [--user-name {user_name}]",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			instance_id, _ := config.InstanceFlag(ctx.Flags()).GetValue()
			wss_url, _ := config.WssUrlFlag(ctx.Flags()).GetValue()
			user_name, _ := config.UserNameFlag(ctx.Flags()).GetValue()
			return doSession(ctx, instance_id, user_name, wss_url)
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

func GetEcsClient(ctx *cli.Context) (*ecs.Client, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		fmt.Errorf("load configuration failed %s", err)
		return nil, fmt.Errorf("load configuration failed %s", err)
	}
	if profile.Mode == "AK" {
		client, err := ecs.NewClientWithAccessKey(profile.RegionId, profile.AccessKeyId, profile.AccessKeySecret)
		return client, err
	} else if profile.Mode == "StsToken" {
		client, err := ecs.NewClientWithStsToken(profile.RegionId, profile.AccessKeyId, profile.AccessKeySecret, profile.StsToken)
		return client, err
	} else if profile.Mode == "CredentialsURI" {
		res, err := getClientByCredentialsURI(profile.CredentialsURI)
		if err != nil {
			return nil, err
		}
		log.GetLogger().Infof("CredentialsURI: %s, %s, %s,%s", profile.RegionId, res.AccessKeyId, res.AccessKeySecret, res.SecurityToken)
		client, err := ecs.NewClientWithStsToken(profile.RegionId, res.AccessKeyId, res.AccessKeySecret, res.SecurityToken)
		return client, err
	} else {
		fmt.Printf("load configuration failed")
		return nil, fmt.Errorf("cound not support current auth mode")
	}

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
	request := ecs.CreateDescribeUserBusinessBehaviorRequest()
	request.Scheme = "https"

	request.StatusKey = "sessionManagerStatus"

	response, err := client.DescribeUserBusinessBehavior(request)
	if err != nil {
		log.GetLogger().Errorln(err)
		fmt.Print(err.Error())
		os.Exit(1)
	}
	if response.StatusValue == "disabled" {
		log.GetLogger().Errorln("session manager is disabled, please enable first")
		fmt.Println("session manager is disabled, please enable first")
		os.Exit(1)
	}

}

func doSession(ctx *cli.Context, instance_id, user_name, wss_url string) error {
	CheckSessionEnabled(ctx)
	var websocket_url string
	if instance_id != "" {
		client, err := GetEcsClient(ctx)
		if err != nil {
			fmt.Print(err.Error())
			log.GetLogger().Errorln(err)
			return fmt.Errorf("get ecs client err:%v", err)
		}

		request := ecs.CreateStartTerminalSessionRequest()
		request.Scheme = "https"

		request.InstanceId = &[]string{instance_id}
		request.Username = user_name

		response, err := client.StartTerminalSession(request)
		if err != nil {
			log.GetLogger().Errorln(err, response)
			fmt.Print(err.Error())
		}
		log.GetLogger().Infof("response is %#v\n", response)
		websocket_url = response.WebSocketUrl
	} else {
		websocket_url = wss_url
	}
	log.GetLogger().Infoln("wss url:", websocket_url)

	url := websocket_url
	url = strings.Replace(url, "sessionid", "sessionId", 1)
	log.GetLogger().Infoln("websocket url:", url)
	client, err := client.NewClient(url, os.Stdin, os.Stdout, false, "", false, config.VerboseFlag(ctx.Flags()).IsAssigned())
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
