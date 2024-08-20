package install

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/service"
)

const (
	ServiceName = "AliyunService"
)

var (
	serviceConfig = &service.Config{
		// 服务显示名称
		Name: ServiceName,
		// 服务名称
		DisplayName: "Aliyun Assist Service",
		// 服务描述
		Description: "Aliyun Assist Service",
	}
)

func ServiceConfig() *service.Config {
	return serviceConfig
}
