package portforward

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/cli"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/client"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/config"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/constant"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/log"
	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/session"

	"github.com/aliyun/aliyun_assist_client/agent/session/plugin/i18n"

	ecsclient "github.com/alibabacloud-go/ecs-20140526/v7/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
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
			idle_timeout, _ := config.IdleTimeoutFlag(ctx.Flags()).GetValue()
			connection_type, _ := config.ConnectionTypeFlag(ctx.Flags()).GetValue()
			var idleTimeout int64
			var err error
			if instance_id == "" {
				fmt.Println("params `instance` is necessary")
				return nil
			}
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
			return doPortForward(ctx, instance_id, local_port, remote_port, service_instance, int32(idleTimeout), connection_type)
		},
	}
	return c
}

func doPortForward(ctx *cli.Context, instance_id string, local_port string, remote_port string, service_instance string, idleTimeout int32, connectionType string) error {
	session.CheckSessionEnabled(ctx)
	var remote_host string
	var err error
	// If remote_port contains ":" then remote host and port are parsed from it,
	// otherwise remote_port is used directly as the remote port.
	if strings.Contains(remote_port, ":") {
		remote_host, remote_port, err = net.SplitHostPort(remote_port)
		if err != nil {
			return fmt.Errorf("parse remote host and port failed: %v", err)
		}
	}
	var websocket_url string
	var session_id string
	if service_instance != "" {
		websocket_url, session_id, err = callComputeNestStartTerminalSession(ctx, service_instance, instance_id, remote_port)
	} else {
		websocket_url, session_id, err = callEcsStartTerminalSession(ctx, instance_id, remote_port, remote_host, connectionType)
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
	fmt.Printf("Port forwarding for SessionId: %s, local port %s, remote port %s:%s\n", session_id, local_port, remote_host, remote_port)
	fmt.Println("Waiting for connections...")
	for {
		local_connect, err := tcp_listener.Accept()
		if err == nil {
			log.GetLogger().Infof("new connection from %s %s\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
			fmt.Printf("new connection from %s %s\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
			go handleConnect(local_connect, url, idleTimeout, ctx)
		}
	}
}

func callEcsStartTerminalSession(ctx *cli.Context, instance_id string, remote_port string, remort_server string, connectionType string) (string, string, error) {
	client, err := session.GetEcsClient(ctx)
	if err != nil {
		fmt.Print(err.Error())
		log.GetLogger().Errorln(err)
		return "", "", fmt.Errorf("get ecs client err:%v", err)
	}
	remote_port_i, _ := strconv.Atoi(remote_port)

	request := ecsclient.StartTerminalSessionRequest{
		RegionId:       client.RegionId,
		InstanceId:     []*string{tea.String(instance_id)},
		PortNumber:     tea.Int32(int32(remote_port_i)),
		TargetServer:   tea.String(remort_server),
		ConnectionType: &connectionType,
	}

	response, err := client.StartTerminalSession(&request)
	if err != nil {
		log.GetLogger().Errorln(err, response)
		fmt.Print(err.Error())
	}
	log.GetLogger().Infof("response is %#v\n", response)
	websocket_url := *response.Body.WebSocketUrl
	sessionId := *response.Body.SessionId
	return websocket_url, sessionId, nil
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

func handleConnect(local_connect net.Conn, url string, idleTimeout int32, ctx *cli.Context) {
	client, err := client.NewClient(url, local_connect, local_connect, true, "", true, config.VerboseFlag(ctx.Flags()).IsAssigned(), idleTimeout)
	if err = client.Loop(); err != nil {
		fmt.Printf("connection[%s %s] err: %v\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String(), err)
		log.GetLogger().Infof("connection[%s %s] err: %v\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String(), err)
	} else {
		fmt.Printf("connection[%s %s] closed\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
		log.GetLogger().Infof("connection[%s %s] closed\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
	}
	if err := local_connect.Close(); err != nil {
		fmt.Printf("close local connection[%s %s] failed, %v\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String(), err)
		log.GetLogger().WithError(err).Errorf("close connection[%s %s] failed\n", local_connect.RemoteAddr().Network(), local_connect.RemoteAddr().String())
	}
}
