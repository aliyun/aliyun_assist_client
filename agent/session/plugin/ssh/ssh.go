// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package ssh

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	client "github.com/aliyun/aliyun_assist_client/agent/session/plugin"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/session"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func NewSshCommand() *cli.Command {

	c := &cli.Command{
		Name: "ssh",
		Short: i18n.T(
			"use ssh proxy devops aliyun ecs instance",
			"使用ssh proxy运维阿里云实例"),
		Usage: "ssh --instance {instance_id}  --port {port}",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			instance_id, _ := config.InstanceFlag(ctx.Flags()).GetValue()
			wss_url, _ := config.WssUrlFlag(ctx.Flags()).GetValue()
			return doSession(ctx, instance_id, wss_url)
		},
	}

	return c
}

func doSession(ctx *cli.Context, instance_id string, wss_url string) error {
	session.CheckSessionEnabled(ctx)
	var websocket_url string
	if instance_id != "" {
		client, err := session.GetEcsClient(ctx)
		if err != nil {
			fmt.Print(err.Error())
			log.GetLogger().Errorln(err)
			return fmt.Errorf("get ecs client err:%v", err)
		}

		request := ecs.CreateStartTerminalSessionRequest()
		request.Scheme = "https"

		request.InstanceId = &[]string{instance_id}

		port_val, _ := config.PortNumberFlag(ctx.Flags()).GetValue()
		if port_val == "" {
			port_val = "22"
		}
		port_i, _ := strconv.Atoi(port_val)
		request.PortNumber = requests.NewInteger(port_i)

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
	client, err := client.NewClient(url, os.Stdin, os.Stdout, false, "", true, config.VerboseFlag(ctx.Flags()).IsAssigned())
	// recvive signal from ssh, avoid killing by ssh
	done := make(chan bool, 1)
	go waitSignals(done)
	if err = client.Loop(); err != nil {
		log.GetLogger().Fatalf("Communication error: %v", err)
	}
	done <- true

	return nil
}
func waitSignals(done chan bool) error {
    sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGHUP,
	)

	select {
	case s := <-sigChan:
		log.GetLogger().Infoln("recv signal: ", s)
		break
	case <- done:
		break
	}
	log.GetLogger().Infoln("waitSignals return")

	return nil
}
