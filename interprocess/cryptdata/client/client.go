package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rodaine/table"
	grpc "google.golang.org/grpc"

	"github.com/aliyun/aliyun_assist_client/agent/cryptdata"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	pb "github.com/aliyun/aliyun_assist_client/interprocess/cryptdata/agrpc"
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
	conn, err := messagebus_client.ConnectWithTimeout(log.GetLogger(), buses.GetCentralEndpoint(false), time.Duration(dialTimeout) * time.Second)
	if err != nil {
		return nil, err
	}

	client := &agentClient{}
	client.Conn = conn
	client.Client = pb.NewAssistAgentClient(client.Conn)
	client.Ctx, client.Cancel = context.WithTimeout(context.Background(), grpcTimeout*time.Second)
	return client, nil
}

func GenRsaKeyPair(keyId string, keyTimeout int) (keyInfo *cryptdata.KeyInfo, errCode int32, err error) {
	var client *agentClient
	errCode = 1
	client, err = newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()
	req := &pb.GenRsaKeyPairReq{
		KeyPairId: keyId,
		Timeout:   int32(keyTimeout),
	}
	var resp *pb.GenRsaKeyPairResp
	resp, err = client.Client.GenRsaKeyPair(client.Ctx, req)
	if err != nil {
		log.GetLogger().Error("Client request GenRsaKeyPair failed: ", err)
		return
	}
	errCode = resp.Status.StatusCode
	if resp.Status.StatusCode != 0 {
		err = errors.New(resp.Status.ErrMessage)
		log.GetLogger().Errorf("GenRsaKeyPair failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	keyInfo = &cryptdata.KeyInfo{
		Id:               resp.KeyInfo.KeyPairId,
		CreatedTimestamp: resp.KeyInfo.CreatedTimestamp,
		ExpiredTimestamp: resp.KeyInfo.ExpiredTimestamp,
		PublicKey:        resp.KeyInfo.PublicKey,
	}
	log.GetLogger().Infof("GenRsaKeyPair success, keyPairId[%s]: ", keyInfo.Id)
	return
}

func RmRsaKeyPair(keyId string) (errCode int32, err error) {
	var client *agentClient
	errCode = 1
	client, err = newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()
	req := &pb.RemoveRsaKeyPairReq {
		KeyPairId: keyId,
	}
	var resp *pb.RemoveRsaKeyPairResp
	resp, err = client.Client.RmRsaKeyPair(client.Ctx, req)
	if err != nil {
		log.GetLogger().Errorf("RmRsaKeyPair failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	errCode = resp.Status.StatusCode
	if resp.Status.StatusCode != 0 {
		err = errors.New(resp.Status.ErrMessage)
		log.GetLogger().Errorf("RmRsaKeyPair failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	log.GetLogger().Infof("RmRsaKeyPair success, keyPairId[%s]: ", keyId)
	return
}

func EncryptText(keyId, plainText string) (cipherText string, errCode int32, err error) {
	var client *agentClient
	errCode = 1
	client, err = newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()
	req := &pb.EncryptReq{
		KeyPairId: keyId,
		PlainText: plainText,
	}
	var resp *pb.EncryptResp
	resp, err = client.Client.EncryptText(client.Ctx, req)
	if err != nil {
		log.GetLogger().Error("Client request EncryptText failed: ", err)
		return
	}
	errCode = resp.Status.StatusCode
	if resp.Status.StatusCode != 0 {
		err = errors.New(resp.Status.ErrMessage)
		log.GetLogger().Errorf("EncryptText failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	cipherText = resp.CipherText
	log.GetLogger().Infof("EncryptText success, keyPairId[%s]: ", keyId)
	return
}
func DecryptText(keyId, cipherText string) (plainText string, errCode int32, err error) {
	var client *agentClient
	errCode = 1
	client, err = newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()
	req := &pb.DecryptReq{
		KeyPairId:  keyId,
		CipherText: cipherText,
	}
	var resp *pb.DecryptResp
	resp, err = client.Client.DecryptText(client.Ctx, req)
	if err != nil {
		log.GetLogger().Error("Client request DecryptText failed: ", err)
		return
	}
	errCode = resp.Status.StatusCode
	if resp.Status.StatusCode != 0 {
		err = errors.New(resp.Status.ErrMessage)
		log.GetLogger().Errorf("DecryptText failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	plainText = resp.PlainText
	log.GetLogger().Infof("DecryptText success, keyPairId[%s]: ", keyId)
	return
}

func CheckKey(keyId string, jsonFlag bool) (output string, errCode int32, err error) {
	var client *agentClient
	errCode = cryptdata.ERR_OTHER_CODE
	defer func() {
		if err != nil && errCode == 0 {
			errCode = cryptdata.ERR_OTHER_CODE
		}
	}()
	client, err = newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()
	req := &pb.CheckKeyReq{
		KeyPairId: keyId,
	}
	var resp *pb.CheckKeyResp
	resp, err = client.Client.CheckKey(client.Ctx, req)
	if err != nil {
		log.GetLogger().Error("Client request DecryptText failed: ", err)
		return
	}
	errCode = resp.Status.StatusCode
	if resp.Status.StatusCode != 0 {
		err = errors.New(resp.Status.ErrMessage)
		log.GetLogger().Errorf("CheckKey failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	if keyId != "" {
		keyInfo := cryptdata.KeyInfo{
				Id: resp.KeyInfos[0].KeyPairId,
				PublicKey: resp.KeyInfos[0].PublicKey,
				CreatedTimestamp: resp.KeyInfos[0].CreatedTimestamp,
				ExpiredTimestamp: resp.KeyInfos[0].ExpiredTimestamp,
			}
		if jsonFlag {
			var content []byte
			if content, err = json.MarshalIndent(keyInfo, "", "\t"); err != nil {
				return
			} else {
				output = string(content)
			}
		} else {
			output = fmt.Sprintf("%s\n%s", keyInfo.Id, keyInfo.PublicKey)
		}
	} else {
		if jsonFlag {
			keyInfos := cryptdata.KeyInfos{}
			for _, k := range resp.KeyInfos {
				keyInfos = append(keyInfos, cryptdata.KeyInfo{
					Id: k.KeyPairId,
					PublicKey: k.PublicKey,
					CreatedTimestamp: k.CreatedTimestamp,
					ExpiredTimestamp: k.ExpiredTimestamp,
				})
			}
			var content []byte
			if content, err = json.MarshalIndent(keyInfos, "", "\t"); err != nil {
				return
			} else {
				output = string(content)
			}
		} else {
			buf := bytes.NewBufferString("")
			tbl := table.New("Id", "CreateTime", "RemainingTime")
			now := time.Now().Unix()
			tbl.WithWriter(buf)
			for _, k := range resp.KeyInfos {
				remainTime := k.ExpiredTimestamp - now
				if remainTime <= 0 {
					continue
				} else {
					tbl.AddRow(k.KeyPairId, k.CreatedTimestamp, remainTime)
				}
			}
			tbl.Print()
			output = buf.String()
		}
	}
	log.GetLogger().Infof("CheckKey success, keyPairId[%s]: ", keyId)
	return
}

func CreateSecretParam(keyId, secretName, cipherText string, timeout int64) (paramInfo *cryptdata.ParamInfo, errCode int32, err error) {
	var client *agentClient
	errCode = 1
	client, err = newClient()
	if err != nil {
		log.GetLogger().Error("Create client failed: ", err)
		return
	}
	defer func() {
		client.Conn.Close()
		client.Cancel()
	}()
	req := &pb.CreateSecretParamReq{
		KeyPairId: keyId,
		CipherText: cipherText,
		SecretName: secretName,
		Timeout: int32(timeout),
	}
	var resp *pb.CreateSecretParamResp
	resp, err = client.Client.CreateSecretParam(client.Ctx, req)
	if err != nil {
		log.GetLogger().Error("Client request CreateSecretParam failed: ", err)
		return
	}
	errCode = resp.Status.StatusCode
	if resp.Status.StatusCode != 0 {
		err = errors.New(resp.Status.ErrMessage)
		log.GetLogger().Errorf("CreateSecretParam failed, keyPairId[%s], StatusCode[%d], errMsg[%s]: ", keyId, resp.Status.StatusCode, resp.Status.ErrMessage)
		return
	}
	paramInfo = &cryptdata.ParamInfo{
		SecretName: resp.SecretParam.SecretName,
		CreatedTimestamp: resp.SecretParam.CreatedTimestamp,
		ExpiredTimestamp: resp.SecretParam.ExpiredTimestamp,
	}
	log.GetLogger().Infof("CreateSecretParam success, keyPairId[%s], secretName[%s]", keyId, secretName)
	return
}