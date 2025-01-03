package metaserver

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
)

func GetAgentProvision(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/global-config/aliyun-assist-agent-provision"
	return fetch(logger, url, requestOptions...)
}

func GetServerCrt(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/global-config/aliyun-assist-server-crt"
	return fetch(logger, url, requestOptions...)
}

func GetServerUrl(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/global-config/aliyun-assist-server-url"
	return fetch(logger, url, requestOptions...)
}

func GetInstanceId(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/meta-data/instance-id"
	return fetch(logger, url, requestOptions...)
}

func GetInstanceName(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/meta-data/instance/instance-name"
	return fetch(logger, url, requestOptions...)
}

func GetRegionId(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/meta-data/region-id"
	return fetch(logger, url, requestOptions...)
}

func GetSecurityCredentials(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/meta-data/ram/security-credentials/"
	return fetch(logger, url, requestOptions...)
}

func GetZoneId(logger logrus.FieldLogger, requestOptions ...httpbase.RequestOption) (string, error) {
	const url = "http://100.100.100.200/latest/meta-data/zone-id"
	return fetch(logger, url, requestOptions...)
}
