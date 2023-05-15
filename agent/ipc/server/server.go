package server

import (
	"context"
	"encoding/base64"

	"github.com/aliyun/aliyun_assist_client/agent/cryptdata"
	pb "github.com/aliyun/aliyun_assist_client/agent/ipc/agrpc"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	grpc "google.golang.org/grpc"
)

type agentServer struct {
	pb.UnimplementedAssistAgentServer
}

var (
	grpcServer *grpc.Server
)

func newRespStatus() *pb.RespStatus {
	return &pb.RespStatus{
		StatusCode: 0,
		ErrMessage: "OK",
	}
}

func newServer() *agentServer {
	s := &agentServer{}
	return s
}

func StartService() {
	lis, err := listen()
	if err != nil {
		log.GetLogger().Errorf("StartService failed to listen: %v", err)
		return
	}
	grpcServer = grpc.NewServer()
	pb.RegisterAssistAgentServer(grpcServer, newServer())
	go func () {
		if err := grpcServer.Serve(lis); err != nil {
			log.GetLogger().Errorf("StartService failed to serve: %v", err)
		}
	}()
}

func (s *agentServer) GenRsaKeyPair(ctx context.Context, req *pb.GenRsaKeyPairReq) (*pb.GenRsaKeyPairResp, error) {
	resp := &pb.GenRsaKeyPairResp{
		Status: newRespStatus(),
		KeyInfo: &pb.KeyInfo{},
	}
	defer func() {
		log.GetLogger().Infof("GenRsaKeyPair keyId[%s] timeout[%d] statusCode[%d] errMsg[%s]", resp.KeyInfo.KeyPairId, req.Timeout, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()
	keyInfo, err := cryptdata.GenRsaKey(req.KeyPairId, int(req.Timeout))
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
		return resp, nil // if return error, client will get a nil resp and lose the error code
	}
	resp.KeyInfo.KeyPairId = keyInfo.Id
	resp.KeyInfo.PublicKey = keyInfo.PublicKey
	resp.KeyInfo.CreatedTimestamp = keyInfo.CreatedTimestamp
	resp.KeyInfo.ExpiredTimestamp = keyInfo.ExpiredTimestamp
	return resp, nil
}

func (s *agentServer) RmRsaKeyPair(ctx context.Context, req *pb.RemoveRsaKeyPairReq) (*pb.RemoveRsaKeyPairResp, error) {
	resp := &pb.RemoveRsaKeyPairResp{
		Status: newRespStatus(),
	}
	defer func() {
		log.GetLogger().Infof("RmRsaKeyPair keyId[%s] statusCode[%d] errMsg[%s]", req.KeyPairId, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()
	err := cryptdata.RemoveRsaKey(req.KeyPairId)
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
	}
	return resp, nil
}

func (s *agentServer) EncryptText(ctx context.Context, req *pb.EncryptReq) (*pb.EncryptResp, error) {
	resp := &pb.EncryptResp{
		Status: newRespStatus(),
	}
	defer func() {
		log.GetLogger().Infof("EncryptText keyId[%s] statusCode[%d] errMsg[%s]", req.KeyPairId, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()
	cipherText, err := cryptdata.EncryptWithRsa(req.KeyPairId, req.PlainText)
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	resp.CipherText = base64.StdEncoding.EncodeToString(cipherText)
	return resp, nil
}

func (s *agentServer) DecryptText(ctx context.Context, req *pb.DecryptReq) (*pb.DecryptResp, error) {
	resp := &pb.DecryptResp{
		Status: newRespStatus(),
	}
	defer func() {
		log.GetLogger().Infof("DecryptText keyId[%s] statusCode[%d] errMsg[%s]", req.KeyPairId, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()
	cipherText, err := base64.StdEncoding.DecodeString(req.CipherText)
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	plainText, err := cryptdata.DecryptWithRsa(req.KeyPairId, cipherText)
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	resp.PlainText = string(plainText)
	return resp, nil
}

func (s *agentServer) CheckKey(ctx context.Context, req *pb.CheckKeyReq) (*pb.CheckKeyResp, error) {
	resp := &pb.CheckKeyResp{
		Status: newRespStatus(),
	}
	defer func() {
		log.GetLogger().Infof("CheckKey keyId[%s] statusCode[%d] errMsg[%s]", req.KeyPairId, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()
	if req.KeyPairId != "" {
		keyInfo, err := cryptdata.CheckKey(req.KeyPairId)
		if err != nil {
			resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
			resp.Status.ErrMessage = err.Error()
			return resp, nil
		}
		resp.KeyInfos = append(resp.KeyInfos, &pb.KeyInfo{
			KeyPairId: keyInfo.Id,
			PublicKey: keyInfo.PublicKey,
			CreatedTimestamp: keyInfo.CreatedTimestamp,
			ExpiredTimestamp: keyInfo.ExpiredTimestamp,
		})
	} else {
		keyList := cryptdata.CheckKeyList()
		for _, keyInfo := range keyList {
			resp.KeyInfos = append(resp.KeyInfos, &pb.KeyInfo{
				KeyPairId: keyInfo.Id,
				PublicKey: keyInfo.PublicKey,
				CreatedTimestamp: keyInfo.CreatedTimestamp,
				ExpiredTimestamp: keyInfo.ExpiredTimestamp,
			})
		}
	}
	return resp, nil
}

func (s *agentServer) CreateSecretParam(ctx context.Context, req *pb.CreateSecretParamReq) (*pb.CreateSecretParamResp, error) {
	resp := &pb.CreateSecretParamResp{
		Status: newRespStatus(),
		SecretParam: &pb.SecretParamInfo{},
	}
	defer func() {
		log.GetLogger().Infof("CreateSecretParam keyId[%s] secretName[%s] timeout[%d] statusCode[%d] errMsg[%s]", req.KeyPairId, req.SecretName, req.Timeout, resp.Status.StatusCode, resp.Status.ErrMessage)
	}()
	cipherText, err := base64.StdEncoding.DecodeString(req.CipherText)
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	paramInfo, err := cryptdata.CreateSecretParam(req.KeyPairId, req.SecretName, int64(req.Timeout), cipherText)
	if err != nil {
		resp.Status.StatusCode = int32(cryptdata.ErrToCode(err))
		resp.Status.ErrMessage = err.Error()
		return resp, nil
	}
	resp.SecretParam.SecretName = paramInfo.SecretName
	resp.SecretParam.CreatedTimestamp = paramInfo.CreatedTimestamp
	resp.SecretParam.ExpiredTimestamp = paramInfo.ExpiredTimestamp
	return resp, nil
}