package portforward

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	client "github.com/aliyun/aliyun_assist_client/agent/session/plugin"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/session"
)

func NewPortForwardCommand() *cli.Command {
	c := &cli.Command{
		Name: "portforward",
		Short: i18n.T(
			"use portforward forward local port to aliyun ecs instance",
			"使用 portforward 将本地端口转发到阿里云实例"),
		Usage: "portforward --instance {instance_id} --localport {local_port} --remoteport {remote_port}",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			instance_id, _ := config.InstanceFlag(ctx.Flags()).GetValue()
			local_port, _ := config.LocalPortFlag(ctx.Flags()).GetValue()
			remote_port, _ := config.RemotePortFlag(ctx.Flags()).GetValue()
			return doPortForward(ctx, instance_id, local_port, remote_port)
		},
	}
	return c
}

func doPortForward(ctx *cli.Context, instance_id string, local_port string, remote_port string) error {
	session.CheckSessionEnabled(ctx)
	var websocket_url string
	ecs_client, err := session.GetEcsClient(ctx)
	if err != nil {
		fmt.Print(err.Error())
		log.GetLogger().Errorln(err)
		return fmt.Errorf("get ecs client err:%v", err)
	}
	request := ecs.CreateStartTerminalSessionRequest()
	request.Scheme = "https"

	request.InstanceId = &[]string{instance_id}
	if remote_port == "" {
		remote_port = "80"
	}
	remote_port_i, _ := strconv.Atoi(remote_port)
	request.PortNumber = requests.NewInteger(remote_port_i)
	response, err := ecs_client.StartTerminalSession(request)
	if err != nil {
		log.GetLogger().Errorln(err, response)
		fmt.Print(err.Error())
		return err
	}
	log.GetLogger().Infof("response is %#v\n", response)
	websocket_url = response.WebSocketUrl
	log.GetLogger().Infoln("wss url:", websocket_url)

	url := websocket_url
	url = strings.Replace(url, "sessionid", "sessionId", 1)
	log.GetLogger().Infoln("websocket url:", url)

	if local_port=="" {
		local_port = "80"
	}
	ip_port := ":"+local_port
	tcp_listener, err := net.Listen("tcp", ip_port)
	if err != nil {
		return fmt.Errorf("start tcp-listener err:%v", err)
	}
	log.GetLogger().Infoln("start tcp-listener, listening ", ip_port)
	fmt.Printf("Port forwarding for SessionId: %s, local port %s, remote port %s\n", response.SessionId, local_port, remote_port)
	fmt.Println("Waiting for connections...")
	for {
		local_connect, err := tcp_listener.Accept()
		if err == nil {
			log.GetLogger().Infof("new connection from %s %s\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
			fmt.Printf("new connection from %s %s\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
			go handleConnect(local_connect, url, ctx)
		}
	}
}

func handleConnect(local_connect net.Conn, url string, ctx *cli.Context) {
	client, err := client.NewClient(url, local_connect, local_connect, true, "", true, config.VerboseFlag(ctx.Flags()).IsAssigned())
	if err = client.Loop(); err != nil {
		fmt.Printf("connection[%s %s] err: %v\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String(), err)
		log.GetLogger().Infof("connection[%s %s] err: %v\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String(), err)
	} else {
		fmt.Printf("connection[%s %s] closed\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
		log.GetLogger().Infof("connection[%s %s] closed\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
	}
}
