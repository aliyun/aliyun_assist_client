package pluginmanager

import (
	"strconv"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

type RemotePlugin struct {
	pui *PluginUpdateInfo

	pi *PluginInfo
}

func (rp *RemotePlugin) Name() string {
	if rp.pi != nil {
		return rp.pi.Name
	}

	return rp.pui.Name
}

func (rp *RemotePlugin) Version() string {
	if rp.pi != nil {
		return rp.pi.Version
	}

	return rp.pui.Version
}

func (rp *RemotePlugin) TimeoutSecs() int {
	if rp.pi != nil {
		var timeout int
		var err error
		if timeout, err = strconv.Atoi(rp.pi.Timeout); err != nil {
			timeout = 60
		}

		return timeout
	}

	return rp.pui.Timeout
}

func (rp *RemotePlugin) Url() string {
	if rp.pi == nil {
		fetchLogger := log.GetLogger().WithFields(logrus.Fields{
			"name": rp.pui.Name,
			"version": rp.pui.Version,
		})

		remotes, err := FetchPackageInfo(fetchLogger, rp.pui.Name, rp.pui.Version, true)
		if err != nil {
			fetchLogger.WithError(err).Error("Failed to fetch plugin information for package URL")
			return ""
		}

		if len(remotes) == 0 {
			fetchLogger.Error("Empty plugin information from remote for package URL")
			return ""
		}

		rp.pi = &remotes[0]
	}

	return rp.pi.Url
}
