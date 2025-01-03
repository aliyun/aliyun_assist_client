package apiserver

import (
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"strconv"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/google/uuid"

	"github.com/aliyun/aliyun_assist_client/agent/hybrid/instance"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/common/networkcategory"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

const (
	// In hybrid instance scenario, domain name <region-id>-axt.aliyuncs.com is used
	// first. If it fails, try domain name <region-id>.axt.aliyuncs.com
	HybridDomainFirst = "-axt.aliyuncs.com"
	HybridDomain      = ".axt.aliyuncs.com"
)

type HybridModeProvider struct{}

type HybridModeHTTPHeadersProvider struct {}

var (
	hybridModeHTTPHeadersProvider = HybridModeHTTPHeadersProvider{}
)

func (*HybridModeProvider) Name() string {
	return "HybridModeProvider"
}

func (p *HybridModeProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
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
		domain := regionId + HybridDomainFirst
		if err := ConnectionDetect(logger, domain); err == nil {
			return domain, nil
		}
		return regionId + HybridDomain, nil
	}
}

func (*HybridModeProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	return hybridModeHTTPHeadersProvider.ExtraHTTPHeaders(logger)
}

func (*HybridModeProvider) RegionId(logger logrus.FieldLogger) (string, error) {
	if !instance.IsHybrid() {
		return "", requester.ErrNotProvided
	}
	if regionId := instance.ReadRegionId(); regionId != "" {
		return regionId, nil
	}
	return "", requester.ErrNotProvided
}

func (*HybridModeHTTPHeadersProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	if !instance.IsHybrid() {
		return nil, requester.ErrNotProvided
	}
	u4 := uuid.New()
	str_request_id := u4.String()

	timestamp := timetool.GetAccurateTime()
	str_timestamp := strconv.FormatInt(timestamp, 10)

	instanceId := instance.ReadInstanceId()
	fingerprint := instance.ReadFingerprint()
	priKey := instance.ReadPriKey()

	input := instanceId + fingerprint + str_timestamp + str_request_id
	output := rsaSign(logger, input, priKey)
	logger.Infoln("Hybrid request:", input, output)

	extraHeaders := map[string]string{
		"x-acs-instance-id": instanceId,
		"x-acs-timestamp":   str_timestamp,
		"x-acs-request-id":  str_request_id,
		"x-acs-signature":   output,
	}

	internal_ip, err := osutil.ExternalIP()
	if err == nil {
		extraHeaders["X-Client-IP"] = internal_ip.String()
	}

	return extraHeaders, nil
}

func getNetworkTypeInHybrid() string {
	return instance.ReadNetworkMode()
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
