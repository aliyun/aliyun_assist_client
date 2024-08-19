package client

import (
	"context"
	"fmt"
	"testing"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/commander/agrpc"
	"github.com/aliyun/aliyun_assist_client/interprocess/messagebus/buses"
	messagebus_server "github.com/aliyun/aliyun_assist_client/interprocess/messagebus/server"
	"github.com/aliyun/aliyun_assist_client/plugin/commander/container/taskerrors"
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
	if req.SubmissionId == "TestRespErrorStatus" {
		return &pb.FinalizeSubmissionStatusResp{
			Status: &pb.RespStatus{
				StatusCode: 1,
				ErrMessage: "TestRespErrorStatus",
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
	if req.SubmissionId == "TestErrorStatusMessage" {
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

func TestUpdateEndpoint(t *testing.T) {
	type args struct {
		endpoint buses.Endpoint
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
		{
			name: "ss",
			args: args{
				endpoint: buses.NewEndpoint("unix", "ss"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, _needRebuildClient, false)
			UpdateEndpoint(tt.args.endpoint)
			assert.Equal(t, _needRebuildClient, true)
			assert.Equal(t, _endpoint.String(), "unix://ss")

		})
	}
}

func TestRegisterCommander(t *testing.T) {
	//1.Register may influence inner agent
	// ctl := gomock.NewController(t)
	// mockRPCAssistAgentServer = NewMockRPCAssistAgentServer()
	if mockRPCAssistAgentServer == nil {
		t.Fatal()
	}
	type args struct {
		logger         logrus.FieldLogger
		handshakeToken string
	}
	UpdateEndpoint(*mockRPCAssistAgentServer.AssistEndPoint)
	t.Run("TestNormalHandshakeToken", func(t *testing.T) {
		err := RegisterCommander(logrus.New(), "RightHandShakeToken")
		fmt.Printf("err:%v", err)
	})
	t.Run("ErrorHandshakeToken", func(t *testing.T) {
		err := RegisterCommander(logrus.New(), "TestErrorHandShakeToken")
		assert.Errorf(t, err, "TestErrorHandShakeToken")
	})
	t.Run("TestErrorType", func(t *testing.T) {
		err := RegisterCommander(logrus.New(), "TestErrorType")
		assert.Errorf(t, err, "SelfTestError")
	})
	// mockRPCAssistAgentServer.AssistServer.GracefulStop()

}

func TestWriteSubmissionOutput(t *testing.T) {
	// mockRPCAssistAgentServer = NewMockRPCAssistAgentServer()
	UpdateEndpoint(*mockRPCAssistAgentServer.AssistEndPoint)
	t.Run("TestErrorStatus", func(t *testing.T) {
		err := WriteSubmissionOutput(logrus.New(), 1, "TestErrorStatusMessage", "TestErrorStatus")
		assert.Errorf(t, err, "TestErrorStatusMessage")
	})
	t.Run("TestReturnError", func(t *testing.T) {
		err := WriteSubmissionOutput(logrus.New(), 1, "TestReturnError", "TestReturnError")
		assert.Errorf(t, err, "TestReturnError")
	})
	t.Run("NormalTest", func(t *testing.T) {
		err := WriteSubmissionOutput(logrus.New(), 1, "NormalTest", "NormalTest")
		fmt.Printf("err:%v", err)
		assert.Equal(t, nil, err)
	})

}

func TestFinalizeSubmissionStatus(t *testing.T) {
	UpdateEndpoint(*mockRPCAssistAgentServer.AssistEndPoint)
	t.Run("ReturnError", func(t *testing.T) {
		err := FinalizeSubmissionStatus(logrus.New(), 1, "TestReturnError", "output", 1, 1, taskerrors.NewContainerNotFoundError())
		assert.Errorf(t, err, "TestReturnError")
	})
	t.Run("TestRespErrorStatus", func(t *testing.T) {
		err := FinalizeSubmissionStatus(logrus.New(), 1, "TestRespErrorStatus", "output", 1, 1, taskerrors.NewContainerNotFoundError())
		assert.Errorf(t, err, "TestRespErrorStatus")
	})
	t.Run("NormalTest", func(t *testing.T) {
		err := FinalizeSubmissionStatus(logrus.New(), 1, "Normal", "output", 1, 1, taskerrors.NewContainerNotFoundError())
		assert.Equal(t, nil, err)
	})
}
