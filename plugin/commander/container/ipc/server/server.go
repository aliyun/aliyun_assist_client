package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/ipc/client"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/model"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskmanager"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"google.golang.org/grpc"
)

type commanderServer struct {
	pb.UnimplementedCommanderServer
	logger logrus.FieldLogger
}

func newRespStatus() *pb.RespStatus {
	return &pb.RespStatus{
		StatusCode: 0,
		ErrMessage: "OK",
	}
}

func RegisterCommanderServer(sr grpc.ServiceRegistrar) {
	pb.RegisterCommanderServer(sr, &commanderServer{
		logger: log.GetLogger().WithField("role", "server"),
	})
}

func (c *commanderServer) Handshake(ctx context.Context, req *pb.HandshakeReq) (*pb.HandshakeResp, error) {
	c.logger.Info("Recv Handshake")
	defer c.logger.Info("Resp Handshake")

	resp := &pb.HandshakeResp{
		Status: newRespStatus(),
	}
	endpoint := buses.Endpoint{}
	if err := endpoint.Parse(req.Endpoint); err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Invalid endpoint: ", err)
		return resp, nil
	}
	client.UpdateEndpoint(endpoint)
	c.logger.Info("endpoint: ", endpoint.String())
	if err := client.RegisterCommander(c.logger, req.HandshakeToken); err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Register commander failed: ", err)
		return resp, nil
	}
	return resp, nil
}

func (c *commanderServer) Ping(ctx context.Context, req *pb.PingReq) (*pb.PingResp, error) {
	// c.logger.Info("Recv Ping")
	// defer c.logger.Info("Resp Ping")
	return &pb.PingResp{
		Status: newRespStatus(),
	}, nil
}

func (c *commanderServer) ConnectFromAgent(ctx context.Context, req *pb.ConnectFromAgentReq) (*pb.ConnectFromAgentResp, error) {
	c.logger.Info("Recv ConnectFromAgent")
	defer c.logger.Info("Resp ConnectFromAgent")

	return &pb.ConnectFromAgentResp{
		Status: newRespStatus(),
		ApiVersion: model.CommanderSupportedApiVersion,
	}, nil
}

func (c *commanderServer) PreCheckScript(ctx context.Context, req *pb.PreCheckScriptReq) (*pb.PreCheckScriptResp, error) {
	c.logger.WithField("submissionId", req.SubmissionId).Infof("Recv PreCheckScript: %+v", req)
	resp := &pb.PreCheckScriptResp{
		Status: newRespStatus(),
	}
	defer c.logger.WithField("submissionId", req.SubmissionId).Infof("Resp PreCheckScript: %+v", resp)

	annotation := make(map[string]string)
	if err := json.Unmarshal([]byte(req.Annotation), &annotation); err != nil {
		return nil, err
	}
	containerId := annotation["containerId"]
	containerName := annotation["containerName"]
	task, err := taskmanager.NewTask(req.SubmissionId, containerId, containerName,
		req.CommandType, req.WorkingDir, req.UserName, int(req.Timeout))
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Create task failed: ", err)
		if taskerr, ok := err.(*taskerrors.TaskError); ok {
			resp.TaskError = &pb.TaskError{
				ErrorCode:    taskerr.ErrorCode,
				ErrorSubCode: taskerr.ErrorSubCode,
				ErrorMessage: taskerr.ErrorMessage,
			}
		}
		return resp, nil
	}
	taskerror := task.PreCheck()
	if taskerror != nil {
		resp.TaskError = &pb.TaskError{
			ErrorCode:    taskerror.ErrorCode,
			ErrorSubCode: taskerror.ErrorSubCode,
			ErrorMessage: taskerror.ErrorMessage,
		}
	}
	resp.ExtraLubanParams = task.ExtraLubanParams()
	return resp, nil
}

func (c *commanderServer) PrepareScript(ctx context.Context, req *pb.PrepareScriptReq) (*pb.PrepareScriptResp, error) {
	c.logger.WithField("submissionId", req.SubmissionId).Infof("Recv PrepareScript: %+v", req)
	resp := &pb.PrepareScriptResp{
		Status: newRespStatus(),
	}
	defer c.logger.WithField("submissionId", req.SubmissionId).Infof("Resp PrepareScript: %+v", resp)

	submissionId := req.SubmissionId
	task, err := taskmanager.LoadTask(submissionId)
	if err == nil {
		err = task.Prepare(req.Content)
	}
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Prepare task failed: ", err)
		resp.TaskError = &pb.TaskError{
			ErrorCode:    err.ErrorCode,
			ErrorSubCode: err.ErrorSubCode,
			ErrorMessage: err.ErrorMessage,
		}
	}
	return resp, nil
}

func (c *commanderServer) SubmitScript(ctx context.Context, req *pb.SubmitScriptReq) (*pb.SubmitScriptResp, error) {
	c.logger.WithField("submissionId", req.SubmissionId).Info("Recv SubmitScript")
	resp := &pb.SubmitScriptResp{
		Status: newRespStatus(),
	}
	defer c.logger.WithField("submissionId", req.SubmissionId).Infof("Resp SubmitScript: %+v", resp)

	submissionId := req.SubmissionId
	task, err := taskmanager.LoadTask(submissionId)
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Submit task failed: ", err)
		resp.TaskError = &pb.TaskError{
			ErrorCode:    err.ErrorCode,
			ErrorSubCode: err.ErrorSubCode,
			ErrorMessage: err.ErrorMessage,
		}
		return resp, nil
	}
	go task.Run()
	return resp, nil
}
func (c *commanderServer) CancelSubmitted(ctx context.Context, req *pb.CancelSubmittedReq) (*pb.CancelSubmittedResp, error) {
	c.logger.WithField("submissionId", req.SubmissionId).Info("Recv CancelSubmitted")
	resp := &pb.CancelSubmittedResp{
		Status: newRespStatus(),
	}
	defer c.logger.WithField("submissionId", req.SubmissionId).Infof("Resp CancelSubmitted: %+v", resp)

	submissionId := req.SubmissionId
	task, err := taskmanager.LoadTask(submissionId)
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Cancel task failed: ", err)
		return resp, nil
	}
	task.Cancel()
	return resp, nil
}
func (c *commanderServer) CleanUpSubmitted(ctx context.Context, req *pb.CleanUpSubmittedReq) (*pb.CleanUpSubmittedResp, error) {
	c.logger.WithField("submissionId", req.SubmissionId).Info("Recv CleanUpSubmitted")
	resp := &pb.CleanUpSubmittedResp{
		Status: newRespStatus(),
	}
	defer c.logger.WithField("submissionId", req.SubmissionId).Infof("Resp CleanUpSubmitted: %+v", resp)

	submissionId := req.SubmissionId
	task, err := taskmanager.LoadTask(submissionId)
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Cancel task failed: ", err)
		return resp, nil
	}
	task.Cleanup()
	return resp, nil
}
func (c *commanderServer) DisposeSubmission(ctx context.Context, req *pb.DisposeSubmissionReq) (*pb.DisposeSubmissionResp, error) {
	c.logger.WithField("submissionId", req.SubmissionId).Info("Recv DisposeSubmission")
	resp := &pb.DisposeSubmissionResp{
		Status: newRespStatus(),
	}
	defer c.logger.WithField("submissionId", req.SubmissionId).Infof("Resp DisposeSubmission: %+v", resp)

	submissionId := req.SubmissionId
	task, err := taskmanager.LoadTask(submissionId)
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = fmt.Sprint("Cancel task failed: ", err)
		return resp, nil
	}
	task.Dispose()
	return resp, nil
}
