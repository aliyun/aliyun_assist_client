package server

import (
	"context"
	"fmt"
	"sort"

	grpc "google.golang.org/grpc"

	"github.com/aliyun/aliyun_assist_client/agent/flagging"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/configure/agrpc"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

type agentServer struct {
	pb.UnimplementedAssistAgentServer
}

func newRespStatus() *pb.RespStatus {
	return &pb.RespStatus{
		StatusCode: 0,
		ErrMessage: "OK",
	}
}

func RegisterAssistAgentServer(sr grpc.ServiceRegistrar) {
	pb.RegisterAssistAgentServer(sr, &agentServer{})
}

type ConfItemList []*pb.ConfItem

func (l ConfItemList) Len() int           { return len(l) }
func (l ConfItemList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
func (l ConfItemList) Less(i, j int) bool { return l[i].Name < l[j].Name }

func (s *agentServer) GetConf(ctx context.Context, req *pb.GetConfReq) (*pb.GetConfResp, error) {
	resp := &pb.GetConfResp{
		Status: newRespStatus(),
	}
	logger := log.GetLogger().WithField("grpc", "GetConf")
	defer func() {
		logger.Infof("runtime[%v], statusCode[%d] errMsg[%s]", req.Runtime, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()

	res := flagging.GetAllConf(logger, req.Runtime)
	for k, v := range res {
		resp.Items = append(resp.Items, &pb.ConfItem{
			Name:  k,
			Value: v,
		})
	}
	sort.Sort(ConfItemList(resp.Items))
	return resp, nil
}

func (s *agentServer) SetConf(ctx context.Context, req *pb.SetConfReq) (*pb.SetConfResp, error) {
	resp := &pb.SetConfResp{
		Status: newRespStatus(),
	}
	logger := log.GetLogger().WithField("grpc", "SetConf")
	defer func() {
		logger.Infof("runtime[%v] crossVer[%v] items[%v], statusCode[%d] errMsg[%s]", req.Runtime, req.CrossVersion, req.Items, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()

	var kvList []string
	for _, item := range req.Items {
		kvList = append(kvList, item.Name)
		kvList = append(kvList, item.Value)
	}
	var err error
	if req.Runtime {
		err = flagging.UpdateConfigRuntime(logger, kvList)
	} else {
		err = flagging.UpdateConfigFile(logger, kvList, req.CrossVersion)
	}
	if err != nil {
		resp.Status.StatusCode = 1
		resp.Status.ErrMessage = err.Error()
	}
	return resp, nil
}

func (s *agentServer) ReloadConf(ctx context.Context, req *pb.ReloadConfReq) (*pb.ReloadConfResp, error) {
	resp := &pb.ReloadConfResp{
		Status: newRespStatus(),
	}
	logger := log.GetLogger().WithField("grpc", "ReloadConf")
	defer func() {
		logger.Infof("statusCode[%d] errMsg[%s]", resp.Status.StatusCode, resp.Status.ErrMessage)
	}()

	flagging.ReloadConfig(logger)
	return resp, nil
}

func GetConfLocal(logger logrus.FieldLogger) []*pb.ConfItem {
	res := []*pb.ConfItem{}
	items := flagging.GetAllConf(logger, false)
	for k, v := range items {
		res = append(res, &pb.ConfItem{
			Name:  k,
			Value: fmt.Sprint(v),
		})
	}
	sort.Sort(ConfItemList(res))
	return res
}

func SetConfLocal(logger logrus.FieldLogger, kvList []string, crossVer bool) error {
	return flagging.UpdateConfigFile(logger, kvList, crossVer)
}
