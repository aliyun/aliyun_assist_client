package server

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	messagebus_server "github.com/aliyun/aliyun_assist_client/interprocess/messagebus/server"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	grpc "google.golang.org/grpc"
)

type MockRPCAssistAgentServer struct {
	// mockServer *MockAssistAgentServer
	AssistServer   *grpc.Server
	AssistEndPoint *buses.Endpoint
}

type mockAssistAgentServer struct {
	pb.UnimplementedAssistAgentServer
}

func (s *mockAssistAgentServer) RegisterCommander(ctx context.Context, req *pb.RegisterCommanderReq) (*pb.RegisterCommanderResp, error) {
	if req.HandshakeToken == "RightHandShakeToken" {
		return &pb.RegisterCommanderResp{
			Status: &pb.RespStatus{
				StatusCode: 0,
			},
		}, nil
	}
	if req.HandshakeToken == "TestErrorHandShakeToken" {
		return &pb.RegisterCommanderResp{
			Status: &pb.RespStatus{
				StatusCode: 1,
				ErrMessage: "TestErrorHandShakeToken",
			},
		}, nil
	}
	if req.HandshakeToken == "TestErrorType" {
		return &pb.RegisterCommanderResp{
			Status: &pb.RespStatus{
				StatusCode: 0,
			},
		}, fmt.Errorf("SelfTestError")
	}

	return &pb.RegisterCommanderResp{
		Status: &pb.RespStatus{
			StatusCode: 0,
		},
	}, nil
}

func (s *mockAssistAgentServer) FinalizeSubmissionStatus(ctx context.Context, req *pb.FinalizeSubmissionStatusReq) (*pb.FinalizeSubmissionStatusResp, error) {
	if req.SubmissionId == "TestErrorStatus" {
		return &pb.FinalizeSubmissionStatusResp{
			Status: &pb.RespStatus{
				StatusCode: 1,
				ErrMessage: "TestErrorStatusMessage",
			},
		}, nil
	}
	if req.SubmissionId == "TestReturnError" {
		return &pb.FinalizeSubmissionStatusResp{
			Status: &pb.RespStatus{
				StatusCode: 0,
			},
		}, fmt.Errorf("TestReturnError")
	}

	return &pb.FinalizeSubmissionStatusResp{
		Status: &pb.RespStatus{
			StatusCode: 0,
		},
	}, nil
}

func (s *mockAssistAgentServer) WriteSubmissionOutput(ctx context.Context, req *pb.WriteSubmissionOutputReq) (*pb.WriteSubmissionOutputResp, error) {
	if req.SubmissionId == "TestErrorStatus" {
		return &pb.WriteSubmissionOutputResp{
			Status: &pb.RespStatus{
				StatusCode: 1,
				ErrMessage: "TestErrorStatusMessage",
			},
		}, nil
	}
	if req.SubmissionId == "TestReturnError" {
		return &pb.WriteSubmissionOutputResp{
			Status: &pb.RespStatus{
				StatusCode: 0,
			},
		}, fmt.Errorf("TestReturnError")
	}

	return &pb.WriteSubmissionOutputResp{
		Status: &pb.RespStatus{
			StatusCode: 0,
		},
	}, nil

}

var (
	mockRPCAssistAgentServer *MockRPCAssistAgentServer
)

func init() {
	mockRPCAssistAgentServer = NewMockRPCAssistAgentServer()
}

func registerAssistAgentServer(sr grpc.ServiceRegistrar) {
	pb.RegisterAssistAgentServer(sr, &mockAssistAgentServer{})
}

func NewMockRPCAssistAgentServer() *MockRPCAssistAgentServer {
	assistEndPoint := buses.GetCentralEndpoint(true)
	tempgRPCServer, err := messagebus_server.ListenAndServe(log.GetLogger(), buses.GetCentralEndpoint(true), nil,
		[]messagebus_server.RegisterFunc{
			registerAssistAgentServer,
		},
	)
	if err != nil {
		fmt.Printf("err:%v", err)
		return nil
	}
	return &MockRPCAssistAgentServer{
		AssistServer:   tempgRPCServer,
		AssistEndPoint: &assistEndPoint,
	}

}

func Test_commanderServer_Handshake(t *testing.T) {
	t.Run("NormalTest", func(t *testing.T) {
		//Right Endpoint with mock assist agent server
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.HandshakeReq{
			Endpoint:       mockRPCAssistAgentServer.AssistEndPoint.String(),
			HandshakeToken: "xxx",
		}
		resp, err := c.Handshake(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(0), resp.Status.StatusCode)
	})
	t.Run("InvalidEndpoint", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.HandshakeReq{
			Endpoint:       "unix:/xx",
			HandshakeToken: "xxx",
		}
		resp, err := c.Handshake(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
	t.Run("RegisterCommanderError", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.HandshakeReq{
			Endpoint:       "unix://xx",
			HandshakeToken: "xxx",
		}
		resp, err := c.Handshake(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
	//Stop assist server
	mockRPCAssistAgentServer.AssistServer.GracefulStop()
}

func Test_commanderServer_PreCheckScript(t *testing.T) {
	//About annotation: annotation should be map[string]string
	//The input param annotation should be
	//No need to init a server or client
	//Container cmd error: No container exec
	t.Run("NormalTest", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		annotation := make(map[string]string)
		annotation["containerId"] = "fad4b98c77b8"
		annotation["containerName"] = "gallant_ramanujan"
		annotationContent, err := json.Marshal(annotation)
		req := &pb.PreCheckScriptReq{
			SubmissionId: "123",
			CommandType:  "unknown",
			WorkingDir:   "unknown",
			Timeout:      20,
			UserName:     "unknown",
			Password:     "unknown",
			Annotation:   string(annotationContent),
		}
		resp, err := c.PreCheckScript(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})

}

func Test_commanderServer_PrepareScript(t *testing.T) {
	t.Run("NormalTest", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.PrepareScriptReq{
			SubmissionId: "xxx",
			Content:      "xxx",
		}
		resp, err := c.PrepareScript(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
}

func Test_commanderServer_SubmitScript(t *testing.T) {
	t.Run("NormalTest", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.SubmitScriptReq{
			SubmissionId: "xxx",
		}
		resp, err := c.SubmitScript(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
}

func Test_commanderServer_CancelSubmitted(t *testing.T) {
	t.Run("NormalTest", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.CancelSubmittedReq{
			SubmissionId: "xx",
		}
		resp, err := c.CancelSubmitted(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
}

func Test_commanderServer_CleanUpSubmitted(t *testing.T) {
	t.Run("NormalTest", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.CleanUpSubmittedReq{
			SubmissionId: "xx",
		}
		resp, err := c.CleanUpSubmitted(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
}

func Test_commanderServer_DisposeSubmission(t *testing.T) {
	t.Run("NormalTest", func(t *testing.T) {
		c := &commanderServer{
			UnimplementedCommanderServer: pb.UnimplementedCommanderServer{},
			logger:                       logrus.New(),
		}
		req := &pb.DisposeSubmissionReq{
			SubmissionId: "xx",
		}
		resp, err := c.DisposeSubmission(context.Background(), req)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(1), resp.Status.StatusCode)
	})
}
