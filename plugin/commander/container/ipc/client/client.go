package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	messagebus_client "github.com/aliyun/aliyun_assist_client/interprocess/messagebus/client"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/endpoint"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

type assistAgentClient struct {
	Conn   *grpc.ClientConn
	Client pb.AssistAgentClient
}

const (
	dialTimeout = 5
)

var (
	_client            *assistAgentClient
	_endpoint          buses.Endpoint // Endpoint of agent server
	_needRebuildClient bool           // _endpoint changed, create new client
	_lock              sync.Mutex
)

func getClient(logger logrus.FieldLogger) (*assistAgentClient, error) {
	_lock.Lock()
	defer _lock.Unlock()

	if _needRebuildClient {
		_needRebuildClient = false
		if _client != nil {
			_client.Conn.Close()
		}
	} else if _client != nil {
		return _client, nil
	}

	conn, err := messagebus_client.ConnectWithTimeout(logger, _endpoint, time.Duration(dialTimeout)*time.Second)
	if err != nil {
		return nil, err
	}
	client := &assistAgentClient{}
	client.Conn = conn
	client.Client = pb.NewAssistAgentClient(client.Conn)

	_client = client
	return _client, nil
}

func UpdateEndpoint(endpoint buses.Endpoint) {
	_lock.Lock()
	defer _lock.Unlock()

	_needRebuildClient = true
	_endpoint = endpoint
}

func RegisterCommander(logger logrus.FieldLogger, handshakeToken string) error {
	logger = logger.WithField("client", "RegisterCommander")
	client, err := getClient(logger)
	if err != nil {
		logger.Error("Create new client failed: ", err)
		return err
	}
	req := &pb.RegisterCommanderReq{
		PluginName: model.CommanderName,
		HandshakeToken: handshakeToken,
		Endpoint:   endpoint.GetEndpoint(false).String(),
		SupportedApiVersion: fmt.Sprintf("[\"%s\"]", model.CommanderSupportedApiVersion),
	}
	resp, err := client.Client.RegisterCommander(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Status.StatusCode != 0 {
		return fmt.Errorf(resp.Status.ErrMessage)
	}
	return nil
}

func WriteSubmissionOutput(logger logrus.FieldLogger, index int, submissionId, output string) error {
	logger = logger.WithField("client", "WriteSubmissionOutput")
	client, err := getClient(logger)
	if err != nil {
		logger.Error("Create new client failed: ", err)
		return err
	}
	req := &pb.WriteSubmissionOutputReq{
		SubmissionId: submissionId,
		Output: &pb.TaskOutput{
			Index:  int32(index),
			Output: output,
		},
	}
	resp, err := client.Client.WriteSubmissionOutput(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Status.StatusCode != 0 {
		return fmt.Errorf(resp.Status.ErrMessage)
	}
	return nil
}

func FinalizeSubmissionStatus(logger logrus.FieldLogger, index int, submissionId, output string,
	exitCode, taskStatus int, taskError *taskerrors.TaskError) error {
	logger = logger.WithField("client", "FinalizeSubmissionStatus")
	client, err := getClient(logger)
	if err != nil {
		logger.Error("Create new client failed: ", err)
		return err
	}
	req := &pb.FinalizeSubmissionStatusReq{
		SubmissionId: submissionId,
		Output: &pb.TaskOutput{
			Index:  int32(index),
			Output: output,
		},
		Result: &pb.TaskRes{
			ExitCode:   int32(exitCode),
			TaskStatus: int32(taskStatus),
		},
	}
	if taskError != nil {
		req.Result.TaskError =  &pb.TaskError{
			ErrorCode: taskError.ErrorCode,
			ErrorSubCode: taskError.ErrorSubCode,
			ErrorMessage: taskError.ErrorMessage,
		}
	}
	resp, err := client.Client.FinalizeSubmissionStatus(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Status.StatusCode != 0 {
		return fmt.Errorf(resp.Status.ErrMessage)
	}
	return nil
}
