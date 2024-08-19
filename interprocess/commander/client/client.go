package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aliyun/aliyun_assist_client/agent/taskengine/taskerrors"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"google.golang.org/grpc"
)

type CommanderClient struct {
	Conn   *grpc.ClientConn
	Client pb.CommanderClient
}

func NewClient(conn *grpc.ClientConn) *CommanderClient {
	client := &CommanderClient{}
	client.Conn = conn
	client.Client = pb.NewCommanderClient(client.Conn)
	return client
}

func (c *CommanderClient) Close() {
	c.Conn.Close()
}

func (c *CommanderClient) Ping(ctx context.Context) (*pb.PingResp, error) {
	req := &pb.PingReq{}
	return c.Client.Ping(ctx, req)
}

func (c *CommanderClient) ConnectFromAgent(logger logrus.FieldLogger, ctx context.Context, supportedApiVersion string) (*pb.ConnectFromAgentResp, error) {
	logger.Info("Request ConnectFromAgent")
	req := &pb.ConnectFromAgentReq{
		SupportedApiVersion: supportedApiVersion,
	}
	return c.Client.ConnectFromAgent(ctx, req)
}

func (c *CommanderClient) PreCheckScript(logger logrus.FieldLogger, ctx context.Context,
	submissionId, commandType string, timeout int,
	workingDir, username, password string, annotation map[string]string) (string, *taskerrors.CommanderError) {
	logger.Info("Request PreCheckScript")
	annotationContent, err := json.Marshal(annotation)
	if err != nil {
		return "", taskerrors.NewIpcRequestFailedError(fmt.Sprintf("marshal annotation fail, %v", err))
	}
	req := &pb.PreCheckScriptReq{
		SubmissionId: submissionId,
		CommandType:  commandType,
		Timeout:      int64(timeout),
		WorkingDir:   workingDir,
		UserName:     username,
		Password:     password,
		Annotation:   string(annotationContent),
	}
	resp, err := c.Client.PreCheckScript(ctx, req)
	if err != nil {
		return "", taskerrors.NewIpcRequestFailedError(fmt.Sprintf("request fail, %v", err))
	}
	if resp.TaskError != nil {
		te := resp.TaskError
		return "", taskerrors.NewCommanderError(te.ErrorCode, te.ErrorSubCode, te.ErrorMessage)
	}
	if resp.Status.StatusCode != 0 {
		return "", taskerrors.NewIpcRequestFailedError(fmt.Sprintf("response status no-zero, %s", resp.Status.ErrMessage))
	}
	return resp.ExtraLubanParams, nil
}

func (c *CommanderClient) PrepareScript(logger logrus.FieldLogger, ctx context.Context,
	submissionId, content string) *taskerrors.CommanderError {
	logger.Info("Request PrepareScript")
	req := &pb.PrepareScriptReq{
		SubmissionId: submissionId,
		Content:      content,
	}
	resp, err := c.Client.PrepareScript(ctx, req)
	if err != nil {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("request fail, %v", err))
	}
	if resp.TaskError != nil {
		te := resp.TaskError
		return taskerrors.NewCommanderError(te.ErrorCode, te.ErrorSubCode, te.ErrorMessage)
	}
	if resp.Status.StatusCode != 0 {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("response status no-zero, %s", resp.Status.ErrMessage))
	}
	return nil
}

func (c *CommanderClient) SubmitScript(logger logrus.FieldLogger, ctx context.Context, submissionId string) *taskerrors.CommanderError {
	logger.Info("Request SubmitScript")
	req := &pb.SubmitScriptReq{
		SubmissionId: submissionId,
	}
	resp, err := c.Client.SubmitScript(ctx, req)
	if err != nil {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("request fail, %v", err))
	}
	if resp.TaskError != nil {
		te := resp.TaskError
		return taskerrors.NewCommanderError(te.ErrorCode, te.ErrorSubCode, te.ErrorMessage)
	}
	if resp.Status.StatusCode != 0 {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("response status no-zero, %s", resp.Status.ErrMessage))
	}
	return nil
}

func (c *CommanderClient) CancelSubmitted(logger logrus.FieldLogger, ctx context.Context, submissionId string) *taskerrors.CommanderError {
	logger.Info("Request CancelSubmitted")
	req := &pb.CancelSubmittedReq{
		SubmissionId: submissionId,
	}
	resp, err := c.Client.CancelSubmitted(ctx, req)
	if err != nil {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("request fail, %v", err))
	}
	if resp.Status.StatusCode != 0 {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("response status no-zero, %s", resp.Status.ErrMessage))
	}
	return nil
}

func (c *CommanderClient) CleanUpSubmitted(logger logrus.FieldLogger, ctx context.Context, submissionId string) *taskerrors.CommanderError {
	logger.Info("Request CleanUpSubmitted")
	req := &pb.CleanUpSubmittedReq{
		SubmissionId: submissionId,
	}
	resp, err := c.Client.CleanUpSubmitted(ctx, req)
	if err != nil {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("request fail, %v", err))
	}
	if resp.Status.StatusCode != 0 {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("response status no-zero, %s", resp.Status.ErrMessage))
	}
	return nil
}

func (c *CommanderClient) DisposeSubmission(logger logrus.FieldLogger, ctx context.Context, submissionId string) *taskerrors.CommanderError {
	logger.Info("Request DisposeSubmission")
	req := &pb.DisposeSubmissionReq{
		SubmissionId: submissionId,
	}
	resp, err := c.Client.DisposeSubmission(ctx, req)
	if err != nil {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("request fail, %v", err))
	}
	if resp.Status.StatusCode != 0 {
		return taskerrors.NewIpcRequestFailedError(fmt.Sprintf("response status no-zero, %s", resp.Status.ErrMessage))
	}
	return nil
}
