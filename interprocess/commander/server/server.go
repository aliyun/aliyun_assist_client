package server

import (
	"context"
	"fmt"

	grpc "google.golang.org/grpc"

	"github.com/aliyun/aliyun_assist_client/agent/commandermanager"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/commander"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	commander_client "github.com/aliyun/aliyun_assist_client/interprocess/commander/client"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type agentServer struct {
	pb.UnimplementedAssistAgentServer
	logger logrus.FieldLogger
}

func newRespStatus() *pb.RespStatus {
	return &pb.RespStatus{
		StatusCode: 0,
		ErrMessage: "OK",
	}
}

func RegisterAssistAgentServer(sr grpc.ServiceRegistrar) {
	pb.RegisterAssistAgentServer(sr, &agentServer{
		logger: log.GetLogger().WithField("role", "server"),
	})
}

func (s *agentServer) RegisterCommander(ctx context.Context, req *pb.RegisterCommanderReq) (*pb.RegisterCommanderResp, error) {
	resp := &pb.RegisterCommanderResp{
		Status: newRespStatus(),
	}
	logger := s.logger.WithField("API", "RegisterCommander")
	logger.Infof("Req: pluginName[%s] endpoint[%s]", req.PluginName, req.Endpoint)
	defer logger.Infof("Resp: %+v", resp.Status)

	// Find registered commander
	p, err1 := commandermanager.GetCommanderWhenHandshake(req.PluginName, req.HandshakeToken)
	if err1 != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Unknown commander: ", err1)
		return resp, nil
	}

	// Create client of commander
	var commander *commander_client.CommanderClient
	endpoint := buses.Endpoint{}
	if err := endpoint.Parse(req.Endpoint); err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Invalid endpoint: ", req.Endpoint, err)
		return resp, nil
	}
	commander, err2 := p.CreateClient(endpoint)
	if err2 != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Client failed: ", err2)
		return resp, nil
	}

	// The current version only supports the initial version of the API,
	// which is hard-coded here.
	supportedApiVersion := fmt.Sprintf("[\"%s\"]", commandermanager.AgentApiVersion)
	cfaResp, reqErr := commander.ConnectFromAgent(logger, context.Background(), supportedApiVersion)
	if reqErr != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Connect from agent failed: ", reqErr)
		return resp, nil
	}
	if cfaResp.Status.StatusCode != 0 {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Connect from agent failed: ", cfaResp.Status.ErrMessage)
		return resp, nil
	}
	// Check if api version match
	if cfaResp.ApiVersion != commandermanager.AgentApiVersion {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprintf("Api versions of commander[%s] agent[%s] not match", cfaResp.ApiVersion, commandermanager.AgentApiVersion)
	}
	p.HandShakeDone(req.HandshakeToken)
	return resp, nil
}
func (s *agentServer) WriteSubmissionOutput(ctx context.Context, req *pb.WriteSubmissionOutputReq) (*pb.WriteSubmissionOutputResp, error) {
	resp := &pb.WriteSubmissionOutputResp{
		Status: newRespStatus(),
	}
	logger := s.logger.WithFields(logrus.Fields{
		"API":          "WriteSubmissionOutput",
		"SubmissionId": req.SubmissionId,
	})
	logger.Infof("Req: index[%d] len[%d]", req.Output.Index, len(req.Output.Output))
	defer logger.Infof("Resp: %+v", resp.Status)

	task, err := commander.FindCommanderTask(req.SubmissionId)
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	task.WriteOutput(int(req.Output.Index), req.Output.Output)
	return resp, nil
}
func (s *agentServer) FinalizeSubmissionStatus(ctx context.Context, req *pb.FinalizeSubmissionStatusReq) (*pb.FinalizeSubmissionStatusResp, error) {
	resp := &pb.FinalizeSubmissionStatusResp{
		Status: newRespStatus(),
	}
	logger := s.logger.WithFields(logrus.Fields{
		"API":          "FinalizeSubmissionStatus",
		"SubmissionId": req.SubmissionId,
	})
	logger.Infof("Req: index[%d] len[%d] exitCode[%d] status[%d] taskErr: %+v", req.Output.Index,
		len(req.Output.Output), req.Result.ExitCode, req.Result.TaskStatus, req.Result.TaskError)
	defer logger.Infof("Resp: %+v", resp.Status)

	task, err := commander.FindCommanderTask(req.SubmissionId)
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	var taskErr *taskerrors.CommanderError
	if req.Result.TaskError != nil {
		te := req.Result.TaskError
		taskErr = taskerrors.NewCommanderError(te.ErrorCode, te.ErrorSubCode, te.ErrorMessage)
	}
	task.WriteOutput(int(req.Output.Index), req.Output.Output)
	task.SetFinalStatus(int(req.Result.ExitCode), int(req.Result.TaskStatus), taskErr)
	return resp, nil
}
