package portforward

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
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
		Usage: "portforward --instance {instance_id} --localport {local_port} --remoteport {remote_port} --service-instance {service_instance_id}",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			instance_id, _ := config.InstanceFlag(ctx.Flags()).GetValue()
			local_port, _ := config.LocalPortFlag(ctx.Flags()).GetValue()
			remote_port, _ := config.RemotePortFlag(ctx.Flags()).GetValue()
			service_instance, _ := config.ServiceInstanceFlag(ctx.Flags()).GetValue()
			if instance_id == "" {
				fmt.Println("params `instance` is necessary")
				return nil
			}
			return doPortForward(ctx, instance_id, local_port, remote_port, service_instance)
		},
	}
	return c
}

func doPortForward(ctx *cli.Context, instance_id string, local_port string, remote_port string, service_instance string) error {
	session.CheckSessionEnabled(ctx)
	if remote_port == "" {
		remote_port = "80"
	}
	var websocket_url string
	var err error
	var session_id string
	if service_instance != "" {
		websocket_url, session_id, err = callComputeNestStartTerminalSession(ctx, service_instance, instance_id, remote_port)
	} else {
		websocket_url, session_id, err = callEcsStartTerminalSession(ctx, instance_id, remote_port)
	}
	if err != nil {
		return fmt.Errorf("start tcp-listener err:%v", err)
	}
	log.GetLogger().Infoln("wss url:", websocket_url)

	url := websocket_url
	url = strings.Replace(url, "sessionid", "sessionId", 1)
	log.GetLogger().Infoln("websocket url:", url)

	if local_port == "" {
		local_port = "80"
	}
	ip_port := ":" + local_port
	tcp_listener, err := net.Listen("tcp", ip_port)
	if err != nil {
		return fmt.Errorf("start tcp-listener err:%v", err)
	}
	log.GetLogger().Infoln("start tcp-listener, listening ", ip_port)
	fmt.Printf("Port forwarding for SessionId: %s, local port %s, remote port %s\n", session_id, local_port, remote_port)
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

func callEcsStartTerminalSession(ctx *cli.Context, instance_id string, remote_port string) (string, string, error) {
	ecs_client, err := session.GetEcsClient(ctx)
	if err != nil {
		fmt.Print(err.Error())
		log.GetLogger().Errorln(err)
		return "", "", fmt.Errorf("get ecs client err:%v", err)
	}
	remote_port_i, _ := strconv.Atoi(remote_port)
	request := ecs.CreateStartTerminalSessionRequest()
	request.Scheme = "https"
	request.InstanceId = &[]string{instance_id}
	request.PortNumber = requests.NewInteger(remote_port_i)
	response, err := ecs_client.StartTerminalSession(request)
	if err != nil {
		log.GetLogger().Errorln(err, response)
		fmt.Print(err.Error())
		return "", "", err
	}
	log.GetLogger().Infof("response is %#v\n", response)
	return response.WebSocketUrl, response.SessionId, nil
}

func callComputeNestStartTerminalSession(ctx *cli.Context, service_instance string, instance_id string, remote_port string) (string, string, error) {

	client, region_id, err := session.GetComputeNestSupplierClient(ctx)
	if err != nil {
		fmt.Print(err.Error())
		log.GetLogger().Errorln(err)
		return "", "", fmt.Errorf("get compute nest supplier client err:%v", err)
	}
	req := requests.NewCommonRequest()
	rep := responses.NewCommonResponse()
	req.Scheme = "HTTPS"
	req.Product = "ComputeNestSupplier"
	req.Version = "2021-05-21"
	req.Domain = "computenestsupplier.cn-hangzhou.aliyuncs.com"
	req.ApiName = "InvokeServiceInstanceOperationAPI"
	req.QueryParams["ServiceInstanceId"] = service_instance
	req.QueryParams["OperationProduct"] = "ecs"
	req.QueryParams["OperationAction"] = "StartTerminalSession"
	req.QueryParams["OperationVersion"] = "2014-05-26"
	type CreateStartTerminalSessionRequest struct {
		InstanceId *[]string        `position:"Query" name:"InstanceId"  type:"Repeated"`
		PortNumber requests.Integer `position:"Query" name:"PortNumber"`
		RegionId   string           `position:"Query" name:"RegionId"`
	}
	remote_port_i, _ := strconv.Atoi(remote_port)

	request := CreateStartTerminalSessionRequest{
		&[]string{instance_id},
		requests.NewInteger(remote_port_i),
		region_id,
	}
	jsonbytes, _ := json.Marshal(request)
	req.QueryParams["OperationParameters"] = string(jsonbytes)
	req.TransToAcsRequest()
	err = client.DoAction(req, rep)
	if err != nil {
		return "", "", err
	}
	var m = make(map[string]string)
	err = json.Unmarshal(rep.GetHttpContentBytes(), &m)
	if err != nil {
		return "", "", err
	}
	response := ecs.CreateStartTerminalSessionResponse()
	err = json.Unmarshal([]byte(m["OperationResults"]), &response)

	return response.WebSocketUrl, response.SessionId, err
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
