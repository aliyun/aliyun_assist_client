package apiserver

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/google/uuid"

	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/common/machineid"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	InternetDomain     = ".axt.aliyuncs.com"
)

type HybridModeProvider struct {}

func (*HybridModeProvider) Name() string {
	return "HybridModeProvider"
}

func (p *HybridModeProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	if domain := os.Getenv("ALIYUN_ASSIST_SERVER_HOST"); domain != "" {
		logger.Info("Get host from env ALIYUN_ASSIST_SERVER_HOST: ", domain)
		return domain, nil
	}

	regionId, err := p.RegionId(logger)
	if regionId == "" || err != nil {
		return "", requester.ErrNotProvided
	}

	// Since region id is determined via hybrid cloud-related function,
	// network category is set to NetworkHybrid
	networkcategory.Set(networkcategory.NetworkHybrid)

	if getNetworkTypeInHybrid() == "vpc" {
		return regionId + IntranetDomain, nil
	} else {
		// Try domain region-axt.aliyuncs.com first,
		// if not success use region.axt.aliyuncs.com
		domain := fmt.Sprintf("%s-%s", regionId, strings.TrimLeft(InternetDomain, "."))
		if err := connectionDetect(logger, domain); err == nil {
			return domain, nil
		}
		return regionId + InternetDomain, nil
	}
}

func (*HybridModeProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	if !IsHybrid() {
		return nil, requester.ErrNotProvided
	}
	u4 := uuid.New()
	str_request_id := u4.String()

	timestamp := timetool.GetAccurateTime()
	str_timestamp := strconv.FormatInt(timestamp, 10)

	var instance_id string
	var path string
	path, _ = pathutil.GetHybridPath()

	content, _ := os.ReadFile(filepath.Join(path, "instance-id"))
	instance_id = string(content)

	mid, _ := machineid.GetMachineID()

	input := instance_id + mid + str_timestamp + str_request_id
	pri_key, _ := os.ReadFile(filepath.Join(path, "pri-key"))
	output := rsaSign(logger, input, string(pri_key))
	logger.Infoln(input, output)

	extraHeaders := map[string]string{
		"x-acs-instance-id": instance_id,
		"x-acs-timestamp": str_timestamp,
		"x-acs-request-id": str_request_id,
		"x-acs-signature": output,
	}

	internal_ip, err := osutil.ExternalIP()
	if err == nil {
		extraHeaders["X-Client-IP"] = internal_ip.String()
	}

	return extraHeaders, nil
}

func (*HybridModeProvider) RegionId(logger logrus.FieldLogger) (string, error) {
	if !IsHybrid() {
		return "", requester.ErrNotProvided
	}

	hybridDir, _ := pathutil.GetHybridPath()
	path := filepath.Join(hybridDir, "region-id")

	if regionIdFile, err := os.Open(path); err == nil {
		if raw, err2 := io.ReadAll(regionIdFile); err2 == nil {
			return strings.TrimSpace(strings.Trim(string(raw), "\r\t\n")), nil
		}
	}
	return "", requester.ErrNotProvided
}

func IsHybrid() bool {
	hybridDir, _ := pathutil.GetHybridPath()
	path := filepath.Join(hybridDir, "instance-id")

	_, err := os.Stat(path)
	return !errors.Is(err, os.ErrNotExist)
}

func getNetworkTypeInHybrid() string {
	hybridDir, _ := pathutil.GetHybridPath()
	path := filepath.Join(hybridDir, "network-mode")

	if networkModeFile, err := os.Open(path); err == nil {
		if raw, err2 := io.ReadAll(networkModeFile); err2 == nil {
			return strings.TrimSpace(strings.Trim(string(raw), "\r\t\n"))
		}
	}
	return ""
}

func rsaSign(logger logrus.FieldLogger, data string, keyBytes string) string {
	w := md5.New()
	io.WriteString(w, data)
	md5_byte := w.Sum(nil)
	value := rsaSignWithMD5(logger, md5_byte, []byte(keyBytes))
	encodeString := base64.StdEncoding.EncodeToString(value)
	return encodeString
}

func rsaSignWithMD5(logger logrus.FieldLogger, data []byte, keyBytes []byte) []byte {
	block, _ := pem.Decode(keyBytes)
	if block == nil {
		logger.Errorln("private key error")
		return []byte{}
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		logger.Errorln("ParsePKCS8PrivateKey err")
		return []byte{}
	}

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.MD5, data)
	if err != nil {
		logger.Errorln("Error from signing")
		return []byte{}
	}

	return signature
}
