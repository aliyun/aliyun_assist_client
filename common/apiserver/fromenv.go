package apiserver

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/timetool"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/google/uuid"

	"github.com/aliyun/aliyun_assist_client/common/machineid"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/requester"
)

type EnvironmentVariableProvider struct{}

func (*EnvironmentVariableProvider) Name() string {
	return "EnvironmentVariableProvider"
}

func (*EnvironmentVariableProvider) CACertificate(logger logrus.FieldLogger) ([]byte, error) {
	certPath := os.Getenv("ALIYUN_ASSIST_CERT_PATH")
	if certPath == "" {
		return nil, requester.ErrNotProvided
	}

	certFile, err := os.Open(certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CA certificate file configured in ALIYUN_ASSIST_CERT_PATH: %w", err)
	}

	pemCerts, err := io.ReadAll(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate file %s configured in ALIYUN_ASSIST_CERT_PATH: %w", certPath, err)
	}

	return pemCerts, nil
}

func (*EnvironmentVariableProvider) ServerDomain(logger logrus.FieldLogger) (string, error) {
	domain := os.Getenv("ALIYUN_ASSIST_SERVER_HOST")
	if domain == "" {
		return "", requester.ErrNotProvided
	}

	return domain, nil
}

func (*EnvironmentVariableProvider) ExtraHTTPHeaders(logger logrus.FieldLogger) (map[string]string, error) {
	domain := os.Getenv("ALIYUN_ASSIST_SERVER_HOST")
	if domain == "" {
		return nil, requester.ErrNotProvided
	}

	u4 := uuid.New()
	str_request_id := u4.String()

	timestamp := timetool.GetAccurateTime()
	str_timestamp := strconv.FormatInt(timestamp, 10)

	var instance_id string
	var path string
	path, _ = pathutil.GetHybridPath()

	content, _ := os.ReadFile(path + "/instance-id")
	instance_id = string(content)

	mid, _ := machineid.GetMachineID()

	input := instance_id + mid + str_timestamp + str_request_id
	pri_key, _ := os.ReadFile(path + "/pri-key")
	output := rsaSign(logger, input, string(pri_key))
	logger.Infoln(input, output)

	extraHeaders := map[string]string{
		"x-acs-instance-id": instance_id,
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
