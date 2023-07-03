package instance

import (
	"os"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/version"
)

const (
	InstanceResourceType = "Ecs.Instance"
)

func GetInstanceInfo() (info *model.InstanceInformation, err error) {
	var i model.InstanceInformation
	i.InstanceId = util.GetInstanceId()
	i.AgentName = "aliyunAssist"
	i.AgentVersion = version.AssistVersion
	i.ComputerName, _ = os.Hostname()
	i.PlatformType, _ = osutil.PlatformType()
	i.PlatformName, _ = osutil.PlatformName()
	if i.PlatformType == "windows" {
		i.PlatformName, _ = osutil.OriginPlatformName()
	}
	i.PlatformVersion, _ = osutil.PlatformVersion()
	i.ResourceType = InstanceResourceType
	ip, _ := osutil.ExternalIP()
	if ip != nil {
		i.IpAddress = ip.String()
	}
	i.RamRole, _ = util.GetRoleNameTtl(time.Duration(5) * time.Hour)
	return &i, nil
}
