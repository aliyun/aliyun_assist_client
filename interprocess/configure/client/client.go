package client

import (
	"context"
	"errors"
	"time"

	grpc "google.golang.org/grpc"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/configure/agrpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	messagebus_client "github.com/aliyun/aliyun_assist_client/interprocess/messagebus/client"
)

type agentClient struct {
	Conn   *grpc.ClientConn
	Client pb.AssistAgentClient
	Cancel context.CancelFunc
	Ctx    context.Context
}

const (
	grpcTimeout = 60
	dialTimeout = 5
)

func newClient() (*agentClient, error) {
	conn, err := messagebus_client.ConnectWithTimeout(log.GetLogger(), buses.GetCentralEndpoint(false), time.Duration(dialTimeout)*time.Second)
	if err != nil {
		return nil, err
	}

	client := &agentClient{}
	client.Conn = conn
	client.Client = pb.NewAssistAgentClient(client.Conn)
	client.Ctx, client.Cancel = context.WithTimeout(context.Background(), grpcTimeout*time.Second)
	return client, nil
}

func GetConf(runtime bool) ([]*pb.ConfItem, error) {
	client, err := newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return nil, err
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()

	req := &pb.GetConfReq{
		Runtime: runtime,
	}
	resp, err := client.Client.GetConf(client.Ctx, req)
	if err != nil {
		log.GetLogger().WithError(err).Error("Client request GetConf failed")
		return nil, err
	}
	if resp.Status.StatusCode != 0 {
		log.GetLogger().Error("GetConf failed, ", resp.Status.ErrMessage)
		return nil, errors.New(resp.Status.ErrMessage)
	}
	return resp.Items, nil
}

func SetConf(kvList []string, runtime, crossVer bool) error {
	client, err := newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return err
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()

	req := &pb.SetConfReq{
		Runtime:      runtime,
		CrossVersion: crossVer,
	}
	for i := 0; i < len(kvList); i += 2 {
		req.Items = append(req.Items, &pb.ConfItem{
			Name:  kvList[i],
			Value: kvList[i+1],
		})
	}
	resp, err := client.Client.SetConf(client.Ctx, req)
	if err != nil {
		log.GetLogger().WithError(err).Error("Client request SetConf failed")
		return err
	}
	if resp.Status.StatusCode != 0 {
		log.GetLogger().Error("SetConf failed, ", resp.Status.ErrMessage)
		return errors.New(resp.Status.ErrMessage)
	}
	return nil
}

func ReloadConf() error {
	client, err := newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return err
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()

	req := &pb.ReloadConfReq{}
	resp, err := client.Client.ReloadConf(client.Ctx, req)
	if err != nil {
		log.GetLogger().WithError(err).Error("Client request ReloadConf failed")
		return err
	}
	if resp.Status.StatusCode != 0 {
		log.GetLogger().Error("ReloadConf failed, ", resp.Status.ErrMessage)
		return errors.New(resp.Status.ErrMessage)
	}
	return nil
}
