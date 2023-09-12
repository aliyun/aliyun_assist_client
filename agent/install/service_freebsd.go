package install

import (
	"github.com/aliyun/aliyun_assist_client/thirdparty/service"
)

const (
	//freebsd的服务名中间不能有"-"，坑死人了
	ServiceName = "aliyun"

	SysvScript = `#!/bin/sh
#
# PROVIDE: {{.Name}}
# REQUIRE: networking syslog
# KEYWORD:
# Add the following lines to /etc/rc.conf to enable the {{.Name}}:
#
# {{.Name}}_enable="YES"
#
. /etc/rc.subr
name="{{.Name}}"
rcvar="{{.Name}}_enable"
command="{{.Path}}"
pidfile="/var/run/$name.pid"
start_cmd="/usr/sbin/daemon -p $pidfile -o /var/log/aliyun-service.log -m 3 -f $command"
load_rc_config $name
run_rc_command "$1"
`
)

var (
	serviceConfig = &service.Config{
		// 服务显示名称
		Name: ServiceName,
		// 服务名称
		DisplayName: "Aliyun Assist Service",
		// 服务描述
		Description: "阿里云助手",
		Option: service.KeyValue{
			"SysvScript": SysvScript,
		},
	}
)

func ServiceConfig() *service.Config {
	return serviceConfig
}
