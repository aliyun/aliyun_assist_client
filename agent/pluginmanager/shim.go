package pluginmanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
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

func (lp *shimLocalPlugin) OSType() string {
	return lp.pi.OSType
}

func (lp *shimLocalPlugin) Architecture() string {
	return lp.pi.Arch
}

func (*ShimManager) FindInstalled(logger logrus.FieldLogger) ([]pluginmodel.LocalPlugin, error) {
	recordeds, err := _findAllInstalledPlugins()
	if err != nil {
		return nil, err
	}
	if len(recordeds) == 0 {
		return nil, nil
	}

	installeds := make([]pluginmodel.LocalPlugin, 0, len(recordeds))
	for _, pluginInfo := range recordeds {
		if !pluginInfo.IsRemoved {
			func(pi PluginInfo){
				installeds = append(installeds, &shimLocalPlugin{
					pi: &pi,
				})
			}(pluginInfo)
		}
	}
	return installeds, nil
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

func (sm *ShimManager) HealthCheck(logger logrus.FieldLogger, strategy pluginmodel.HealthCheckStrategy) ([]pluginmodel.PluginStatus, error) {
	switch strategy {
	case pluginmodel.HealthCheckByScan:
		return sm.healthCheckByScan(logger)
	case pluginmodel.HealthCheckByPull:
		return sm.healthCheckByPull(logger)
	default:
		// SHOULD NEVER RUN INTO THIS BRANCH
		return nil, nil
	}
}

func (*ShimManager) healthCheckByScan(logger logrus.FieldLogger) ([]pluginmodel.PluginStatus, error) {
	// 1.检查插件列表，如果没有插件就不需要健康检查
	installeds, err := _findAllInstalledPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load installed plugins: %w", err)
	}
	if len(installeds) == 0 {
		return nil, nil
	}

	statuses := make([]pluginmodel.PluginStatus, 0, len(installeds))
	persistPluginCount := 0
	pluginInfoMap := make(map[string]*PluginInfo)
	for _, pluginInfo := range installeds {
		if pluginInfo.IsRemoved {
			continue
		}

		pluginInfoMap[pluginInfo.Name] = &pluginInfo
		if pluginInfo.PluginType() == PLUGIN_ONCE {
			statuses = append(statuses, pluginmodel.PluginStatus{
				Name:    pluginInfo.Name,
				Status:  pluginmodel.ONCE_INSTALLED,
				Version: pluginInfo.Version,
			})
		} else if pluginInfo.PluginType() == PLUGIN_PERSIST {
			persistPluginCount += 1
		}
	}
	if persistPluginCount > 0 {
		// 调用acs-plugin-manager模块的 status接口，批量获取常驻插件状态（包括已删除的常驻插件）
		mixedOutput := bytes.Buffer{}
		cmd := "acs-plugin-manager"
		arguments := []string{"--status"}
		_, _, err = syncRunKillGroup("", cmd, arguments, &mixedOutput, &mixedOutput, 120)
		if err != nil {
			logger.Errorf("pluginHealthCheckScan: cmd run err: %s, cmd[%s %s] output[%s]", err.Error(), cmd, strings.Join(arguments, " "), mixedOutput.String())
			return nil, err
		}
		content := mixedOutput.Bytes()
		pluginStatusList := []pluginmodel.PluginStatus{}
		if err := json.Unmarshal(content, &pluginStatusList); err != nil {
			logger.Errorf("pluginHealthCheckScan: json.Unmarshal pluginStatusList error: %s, content: %s", err.Error(), string(content))
		}
		if len(pluginStatusList) == 0 {
			logger.Infof("pluginHealthCheckScan: there is no persist plugin, content[%s]", string(content))
		}

		for _, pluginInfo := range pluginStatusList {
			if pluginInfo.Status == pluginmodel.REMOVED {
				continue
			}
			pluginStatus := pluginmodel.PluginStatus{
				Name:    pluginInfo.Name,
				Version: pluginInfo.Version,
				Status:  pluginInfo.Status,
			}
			if pluginInfo.Status != pluginmodel.PERSIST_RUNNING && pluginInfo.Status != pluginmodel.REMOVED {
				// // 状态异常的常驻插件本次不上报，acs-plugin-manager调用--start拉起后会单独上报该插件的状态
				logger.Warnf("plugin[%s] is not running, try to start it", pluginInfo.Name)
				go func(pluginName string, mp map[string]*PluginInfo) {
					randSleep := rand.Intn(10 * 1000)
					time.Sleep(time.Duration(randSleep) * time.Millisecond)
					command := "acs-plugin-manager"
					arguments := []string{"-e", "--local", "-P", pluginName, "-p", "--start"}
					timeout := 60
					if pluginInfoPtr, ok := mp[pluginName]; ok && pluginInfoPtr.Timeout != "" {
						if t, err := strconv.Atoi(pluginInfoPtr.Timeout); err == nil {
							timeout = t
						}
					}
					syncRunKillGroup("", command, arguments, nil, nil, timeout)
				}(pluginInfo.Name, pluginInfoMap)
			} else {
				// 状态正常的常驻插件进行上报
				statuses = append(statuses, pluginStatus)
			}
		}
	}

	return statuses, nil
}

func (*ShimManager) healthCheckByPull(logger logrus.FieldLogger) ([]pluginmodel.PluginStatus, error) {
	now := time.Now().Unix()

	// 1.检查插件列表，如果没有插件就不需要健康检查
	installeds, err := _findAllInstalledPlugins()
	if err != nil {
		return nil, fmt.Errorf("failed to load installed plugins: %w", err)
	}
	if len(installeds) == 0 {
		return nil, nil
	}

	pluginDir, err := pathutil.GetPluginPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve global plugin directory: %w", err)
	}

	statuses := make([]pluginmodel.PluginStatus, 0, len(installeds))
	for _, pluginInfo := range installeds {
		if pluginInfo.IsRemoved {
			continue
		}
		if pluginInfo.PluginType() != PLUGIN_PERSIST {
			continue
		}

		// 常驻型插件且未被删除：检查并读取插件目录下的heartbeat文件
		heartbeatPath := filepath.Join(pluginDir, pluginInfo.Name, pluginInfo.Version, "heartbeat")
		if fileutil.CheckFileIsExist(heartbeatPath) {
			content, err := os.ReadFile(heartbeatPath)
			if err != nil {
				logger.Errorf("pluginHealthCheckPull: Read heartbeat file err, heartbeat[%s], err: %s", heartbeatPath, err.Error())
				continue
			}
			timestampStr := strings.TrimSpace(string(content))
			timestamp, err := strconv.ParseInt(timestampStr, 10, 0)
			if err != nil {
				logger.Errorf("pluginHealthCheckPull: Parse heartbeat file err, heartbeat[%s], content[%s] err: %s", heartbeatPath, timestampStr, err.Error())
				continue
			}
			status := pluginmodel.PERSIST_RUNNING
			if now-timestamp > int64(pluginInfo.HeartbeatInterval+5) {
				status = pluginmodel.PERSIST_FAIL
			}

			statuses = append(statuses, pluginmodel.PluginStatus{
				Name:    pluginInfo.Name,
				Status:  status,
				Version: pluginInfo.Version,
			})
		}
	}
	return statuses, nil
}
