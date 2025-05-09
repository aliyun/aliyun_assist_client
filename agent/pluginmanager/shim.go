package pluginmanager

import (
	"bytes"
	"strconv"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmodel"
)

type shimLocalPlugin struct {
	pi *PluginInfo
}

type ShimManager struct {}

func (lp *shimLocalPlugin) Name() string {
	return lp.pi.Name
}

func (lp *shimLocalPlugin) Version() string {
	return lp.pi.Version
}

func (*ShimManager) FindUpgradable(logger logrus.FieldLogger) ([]pluginmodel.LocalPlugin, error) {
	recordeds, err := _findAllInstalledPlugins()
	if err != nil {
		logger.WithError(err).Error("pluginUpdateCheck fail: loadPlugins fail")
		return nil, err
	}
	if len(recordeds) == 0 {
		logger.Info("pluginUpdateCheck cancel: there is no plugins")
		return nil, nil
	}

	upgradables := make([]pluginmodel.LocalPlugin, 0, len(recordeds))
	for _, pluginInfo := range recordeds {
		if (pluginInfo.PluginType() == PLUGIN_PERSIST || pluginInfo.PluginType() == PLUGIN_COMMANDER) && !pluginInfo.IsRemoved {
			func(pi PluginInfo){
				upgradables = append(upgradables, &shimLocalPlugin{
					pi: &pi,
				})
			}(pluginInfo)
		}
	}
	return upgradables, nil
}

func (*ShimManager) Update(logger logrus.FieldLogger, target pluginmodel.RemotePlugin) error {
	handler := getUpdateHandler()
	if handler != nil && handler(target.Name(), target.Version()) {
		return nil
	}

	command := "acs-plugin-manager"
	arguments := []string{"--exec", "-P", target.Name(), "-n", target.Version(), "-p", "--upgrade"}
	mixedOutput := bytes.Buffer{}
	exitCode, status, err := syncRunKillGroup("", command, arguments, &mixedOutput, &mixedOutput, target.TimeoutSecs()+5)
	output := mixedOutput.String()
	if len(output) > 1024 {
		output = output[:1024]
	}
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	}
	metrics.GetPluginUpdateEvent(
		"name", target.Name(),
		"version", target.Version(),
		"exitCode", strconv.Itoa(exitCode),
		"status", strconv.Itoa(status),
		"errMsg", errMsg,
		"output", output,
	).ReportEvent()

	updatedLogger := logger.WithFields(logrus.Fields{
		"name": target.Name(),
		"version": target.Version(),
		"exitcode": exitCode,
		"status": status,
		"output": output,
	})
	if err != nil {
		updatedLogger.WithError(err).Errorf("pluginUpdateCheck: failed to update plugin")
	} else {
		updatedLogger.Info("pluginUpdateCheck: plugin updated successfully")
	}

	return nil
}
