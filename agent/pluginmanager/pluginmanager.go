package pluginmanager

import (
	"encoding/json"
	"errors"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmodel"
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
)

/*
健康检查：两种并行的周期上报插件状态的方式
	1. 调用 acs-plugin-manager --status，通过常驻插件的 --status 接口获得插件状态
	2. 读取常驻插件目录下的 heartbeat 文件获得常驻插件的心跳时间戳，通过心跳时间戳判断插件状态
		通过heartbeat判断插件状态后可以进行懒上报，只在插件状态变化时进行上报；默认每次都上报，受服务端返回的reportType字段控制
		lastPluginStatusRecord 用来记录上一次各插件的状态


常驻插件升级：
	1. 读取installed_plugins文件获得所有插件信息，检查是否有已安装的常驻插件
	2. 常驻插件的版本信息上报服务端
	3. 遍历服务端响应的升级列表
		1. 执行 `acs-plugin-manager --exec --plugin <pluginName> --pluginVersion <pluginVersion> --params --upgrade`。升级插件
*/

var (
	pluginHealthScanInterval  = 15 * 60 // 通过常驻插件的 --status 接口获取插件状态的时间间隔
	pluginHealthPullInterval  = 5 * 60  // 通过读取常驻插件心跳时间戳判断插件状态的时间间隔
	avoidTime                 = 60      // 单位秒， pluginHealthPull 与 pluginHealthScan 的相距时间需要大于avoidTime，避免短时间内重复上报
	lastPluginHealthCheckTime int64     // 记录上一次插件状态检查的时间，避免两种插件状态上报方式同时上报
	pluginHealthCheckTimeMut  sync.Mutex
	lazyReport                bool
	lastPluginStatusRecord    = map[string]string{}

	// pluginUpdateCheckIntervalSeconds 常驻插件检查升级的间隔时间
	pluginUpdateCheckInterval = 30 * 60

	pluginListReportInterval = 3600 * 24

	LocalManager pluginmodel.LocalManager = &ShimManager{}
)

type UpdateHandler func(name, version string) bool

var (
	pluginHealthScanTimer *timermanager.Timer
	pluginHealthPullTimer *timermanager.Timer
	pluginListReportTimer *timermanager.Timer
	pluginUpdateTimer     *timermanager.Timer

	updateHandler     UpdateHandler
	updateHandlerLock sync.Mutex
)

func InitPluginCheckTimer() {
	var err error
	loadIntervalConf()
	timerManager := timermanager.GetTimerManager()
	if pluginHealthScanTimer, err = timerManager.CreateTimerInSeconds(pluginHealthCheckScan, pluginHealthScanInterval); err != nil {
		log.GetLogger().Error("InitPluginCheckTimer: pluginHealthScanTimer err: ", err.Error())
	} else {
		go func() {
			// shuffle timer in 1 minutes
			mills := rand.Intn(60 * 1000)
			time.Sleep(time.Duration(mills) * time.Millisecond)
			if _, err = pluginHealthScanTimer.Run(); err != nil {
				log.GetLogger().Error("InitPluginCheckTimer: pluginHealthScanTimer run err: ", err.Error())
			}
		}()
	}
	if pluginHealthPullTimer, err = timerManager.CreateTimerInSeconds(pluginHealthCheckPull, pluginHealthPullInterval); err != nil {
		log.GetLogger().Error("InitPluginCheckTimer: pluginHealthPullTimer err: ", err.Error())
	} else {
		go func() {
			mills := rand.Intn(60 * 1000)
			// make sure that pluginHealthCheckScan is earlier than pluginHealthPullTimer
			time.Sleep(time.Duration(60000+mills) * time.Millisecond)
			if _, err = pluginHealthPullTimer.Run(); err != nil {
				log.GetLogger().Error("InitPluginCheckTimer: pluginHealthPullTimer run err: ", err.Error())
			}
		}()
	}
	if pluginUpdateTimer, err = timerManager.CreateTimerInSeconds(pluginUpdateCheck, pluginUpdateCheckInterval); err != nil {
		log.GetLogger().Error("InitPluginCheckTimer: pluginUpdateTimer err: ", err.Error())
	} else {
		go func() {
			if _, err = pluginUpdateTimer.Run(); err != nil {
				log.GetLogger().Error("InitPluginCheckTimer: pluginUpdateTimer run err: ", err.Error())
			}
		}()
	}
	if pluginListReportTimer, err = timerManager.CreateTimerInSeconds(pluginLocalListReport, pluginListReportInterval); err != nil {
		log.GetLogger().Error("InitPluginCheckTimer: pluginListReportTimer err: ", err.Error())
	} else {
		go func() {
			mills := rand.Intn(60 * 1000)
			time.Sleep(time.Duration(180000+mills) * time.Millisecond)
			log.GetLogger().Info("InitPluginCheckTimer:pluginLocalListReport timer run")
			_, err = pluginListReportTimer.Run()
			if err != nil {
				log.GetLogger().Error("InitPluginCheckTimer:pluginLocalListReport timer run err: ", err.Error())
			}
		}()
	}
}

func SetUpdateHandler(h UpdateHandler) {
	updateHandlerLock.Lock()
	defer updateHandlerLock.Unlock()

	updateHandler = h
}

func getUpdateHandler() UpdateHandler {
	updateHandlerLock.Lock()
	defer updateHandlerLock.Unlock()

	return updateHandler
}

func pluginHealthCheckScan() {
	pluginHealthCheckTimeMut.Lock()
	lastPluginHealthCheckTime = time.Now().Unix()
	pluginHealthCheckTimeMut.Unlock()
	log.GetLogger().Info("pluginHealthCheckScan: start")

	// 2.将插件状态发送给服务端
	statuses, err := LocalManager.HealthCheck(log.GetLogger(), pluginmodel.HealthCheckByScan)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckScan: error encountered when scanning plugins that need to report status")
		return
	}
	if len(statuses) == 0 {
		log.GetLogger().Info("pluginHealthCheckScan: there is no plugin need report status")
		return
	}

	pluginStatusRequest := _truncateStatusRequestFields(&PluginStatusResquest{
		Plugin: statuses,
	})
	requestPayloadBytes, err := json.Marshal(pluginStatusRequest)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckScan: pluginStatusList marshal err: " + err.Error())
		return
	}
	requestPayload := string(requestPayloadBytes)
	url := util.GetPluginHealthService()
	resp, err := util.HttpPost(url, requestPayload, "")

	for i := 0; i < 3 && err != nil; i++ {
		log.GetLogger().Infof("pluginHealthCheckScan: upload pluginStatusList fail, need retry: %s", requestPayload)
		time.Sleep(time.Duration(2) * time.Second)
		resp, err = util.HttpPost(url, requestPayload, "")
	}
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckScan: post pluginStatusList fail")
		return
	}
	pluginStatusResp, err := parsePluginHealthCheck(resp)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("pluginHealthCheckScan: parse PluginStatusResponse from resp fail: %s", resp)
		return
	}
	// 设置下次状态检查周期
	if pluginStatusResp.PullInterval > 0 && pluginStatusResp.PullInterval != pluginHealthPullInterval {
		pluginHealthPullInterval = pluginStatusResp.PullInterval
	}
	if pluginStatusResp.ScanInterval > 0 && pluginStatusResp.ScanInterval != pluginHealthScanInterval {
		pluginHealthScanInterval = pluginStatusResp.ScanInterval
	}
	if err := refreshTimer(pluginHealthScanTimer, pluginHealthScanInterval); err != nil {
		log.GetLogger().Errorf("pluginHealthCheckScan: refresh pluginHealthScanTimer nextInterval [%d] second failed: %s", pluginHealthScanInterval, err.Error())
	} else {
		log.GetLogger().Infof("pluginHealthCheckScan: refresh pluginHealthScanTimer nextInterval [%d] second", pluginHealthScanInterval)
	}

	if pluginStatusResp.ReportType == NORMAL_REPORT && lazyReport {
		lazyReport = false
		log.GetLogger().Info("pluginHealthCheckScan: lazyReport switch to [off]")
	} else if pluginStatusResp.ReportType == LAZY_REPORT && !lazyReport {
		lazyReport = true
		log.GetLogger().Info("pluginHealthCheckScan: lazyReport switch to [on]")
	}
	// if flowReport {
	// 	// 有拉起插件的动作，需要重置pluginHealthPullTimer以便及时向服务端更新拉起后的状态
	// 	// 但是如果interval太晚（晚于pluginHealthPullInterval 或者 pluginHealthScanInterval）就不需要重置pluginHealthPullTimer了
	// 	interval := 60
	// 	if pluginStatusResp.RefreshInterval > 0 {
	// 		interval = pluginStatusResp.RefreshInterval
	// 	}
	// 	if interval < pluginHealthPullInterval && interval < pluginHealthScanInterval {
	// 		pluginHealthPullTimer.Reset(time.Duration(interval) * time.Second)
	// 	}
	// }
	log.GetLogger().Info("pluginHealthCheckScan success")
}

func pluginHealthCheckPull() {
	pluginHealthCheckTimeMut.Lock()
	lastTime := lastPluginHealthCheckTime
	pluginHealthCheckTimeMut.Unlock()
	now := time.Now().Unix()
	needWait := int64(avoidTime) - (now - lastTime)
	if needWait > 0 {
		log.GetLogger().Infof("pluginHealthCheckPull: last pluginHealthCheckScan started [%d] seconds ago, need wait [%d] second for avoidTime[%d]", now-lastTime, needWait, avoidTime)
		time.Sleep(time.Duration(needWait) * time.Second)
		now = time.Now().Unix()
	}
	remainTime := lastTime + int64(pluginHealthScanInterval) - now
	if remainTime < int64(avoidTime) {
		log.GetLogger().Infof("pluginHealthCheckPull: next pluginHealthCheckScan will start in [%d] seconds, less than avoidTime[%d], so cancel this pluginHealthCheckPull", remainTime, avoidTime)
		return
	}
	log.GetLogger().Info("pluginHealthCheckPull: start")

	statuses, err := LocalManager.HealthCheck(log.GetLogger(), pluginmodel.HealthCheckByPull)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckPull: error encountered when pulling heartbeat of plugins")
		return
	}
	if len(statuses) == 0 {
		log.GetLogger().Info("pluginHealthCheckPull: there is no persist plugin with heartbeat")
		return
	}

	curPluginStatusRecord := make(map[string]string, len(statuses))
	for _, ps := range statuses {
		curPluginStatusRecord[ps.Name] = ps.Status
	}
	willReport := true
	if lazyReport {
		// 如果lazyReport为true，对比本次上报的插件状态和上次的插件状态是否一致，如果不一致才上报
		willReport = false
		// 数量是否一致
		if len(curPluginStatusRecord) != len(lastPluginStatusRecord) {
			willReport = true
		} else {
			for k, v := range curPluginStatusRecord {
				// 插件项目是否一致
				if _, ok := lastPluginStatusRecord[k]; !ok {
					willReport = true
					break
				}
				// 同一插件的状态是否一致
				if v != lastPluginStatusRecord[k] {
					willReport = true
					break
				}
			}
		}
	}

	if !willReport {
		return
	}
	lastPluginStatusRecord = curPluginStatusRecord

	pluginStatusRequest := _truncateStatusRequestFields(&PluginStatusResquest{
		Plugin: statuses,
	})
	requestPayloadBytes, err := json.Marshal(pluginStatusRequest)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckPull fail: pluginStatusList marshal fail")
		return
	}
	requestPayload := string(requestPayloadBytes)
	url := util.GetPluginHealthService()
	resp, err := util.HttpPost(url, requestPayload, "")
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckPull fail: post pluginStatusList fail")
		return
	}
	pluginStatusResp, err := parsePluginHealthCheck(resp)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("pluginHealthCheckPull fail: parse PluginStatusResponse from resp fail: %s", resp)
		return
	}

	// 设置下次状态检查周期
	if pluginStatusResp.PullInterval > 0 {
		pluginHealthPullInterval = pluginStatusResp.PullInterval
	}
	if err := refreshTimer(pluginHealthPullTimer, pluginHealthPullInterval); err != nil {
		log.GetLogger().Errorf("pluginHealthCheckPull: refresh pluginHealthPullTimer nextInterval [%d] second failed: %s", pluginHealthPullInterval, err.Error())
	} else {
		log.GetLogger().Infof("pluginHealthCheckPull: refresh pluginHealthPullTimer nextInterval [%d] second", pluginHealthPullInterval)
	}
	if pluginStatusResp.ScanInterval > 0 {
		pluginHealthScanInterval = pluginStatusResp.ScanInterval
	}
	// pluginStatusResp.ReportType 代表是否开启懒上报
	if pluginStatusResp.ReportType == NORMAL_REPORT && lazyReport {
		lazyReport = false
		log.GetLogger().Info("pluginHealthCheckPull: lazyReport switch to [off]")
	} else if pluginStatusResp.ReportType == LAZY_REPORT && !lazyReport {
		lazyReport = true
		log.GetLogger().Info("pluginHealthCheckPull: lazyReport switch to [on]")
	}
	log.GetLogger().Info("pluginHealthCheckPull success")
}

func pluginUpdateCheck() {
	log.GetLogger().Info("pluginUpdateCheck start")

	// get upgradable plugin list
	upgradables, err := LocalManager.FindUpgradable(log.GetLogger())
	if err != nil || len(upgradables) == 0 {
		log.GetLogger().WithError(err).Info("pluginUpdateCheck cancel: there is no persist plugin")
		return
	}

	// request for update check
	osType := osutil.GetOsType()
	arch, _ := GetArch()
	updateChecks := make([]PluginUpdateCheck, 0, len(upgradables))
	for _, u := range upgradables {
		updateChecks = append(updateChecks, PluginUpdateCheck{
			Name: u.Name(),
			Version: u.Version(),
		})
	}
	pluginUpdateCheckRequest := PluginUpdateCheckRequest{
		Os:     osType,
		Arch:   arch,
		Plugin: updateChecks,
	}

	requestPayloadBytes, err := json.Marshal(pluginUpdateCheckRequest)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginUpdateCheck fail: pluginUpdateCheckRequest marshal fail")
		return
	}
	requestPayload := string(requestPayloadBytes)
	url := util.GetPluginUpdateCheckService()
	resp, err := util.HttpPost(url, requestPayload, "")
	for i := 0; i < 2 && err != nil; i++ {
		log.GetLogger().Infof("request updateCheck fail, need retry: %s", requestPayload)
		time.Sleep(time.Duration(3) * time.Second)
		resp, err = util.HttpPost(url, requestPayload, "")
	}
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginUpdateCheck fail: post pluginStatusList fail")
		return
	}
	// update plugins
	var pluginUpdateCheckResp PluginUpdateCheckResponse
	pluginUpdateCheckResp, err = parsePluginUpdateCheck(resp)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("pluginUpdateCheck fail: parse pluginUpdateInfo from resp fail: %s", resp)
		return
	}

	for _, plugin := range pluginUpdateCheckResp.Plugin {
		updateOneLogger := log.GetLogger().WithFields(logrus.Fields{
			"name": plugin.Name,
			"version": plugin.Version,
			"timeout": plugin.Timeout,
		})

		err := LocalManager.Update(updateOneLogger, &RemotePlugin{pui: &plugin})
		if err != nil {
			updateOneLogger.WithError(err).Error("pluginUpdateCheck: failed to update one plugin to new version in specified time")
		}
	}
	log.GetLogger().Infof("pluginUpdateCheck done, updated [%d] plugins", len(pluginUpdateCheckResp.Plugin))
	if pluginUpdateCheckResp.NextInterval > 0 {
		pluginUpdateCheckInterval = pluginUpdateCheckResp.NextInterval
	}
	if err := refreshTimer(pluginUpdateTimer, pluginUpdateCheckInterval); err != nil {
		log.GetLogger().Errorf("pluginUpdateCheck: refresh pluginUpdateTimer nextInterval [%d] second failed: %s", pluginUpdateCheckInterval, err.Error())
	} else {
		log.GetLogger().Errorf("pluginUpdateCheck: refresh pluginUpdateTimer nextInterval [%d] second", pluginUpdateCheckInterval)
	}
}

func pluginLocalListReport() {
	log.GetLogger().Info("pluginLocalListReport: start")
	installeds, err := LocalManager.FindInstalled(log.GetLogger())
	if err != nil || len(installeds) == 0 {
		log.GetLogger().WithError(err).Error("pluginLocalListReport: no local installed plugin(s) loaded")
		return
	}

	nameList := []string{}
	versionList := []string{}
	osList := []string{}
	archList := []string{}
	for _, p := range installeds {
		nameList = append(nameList, p.Name())
		versionList = append(versionList, p.Version())
		osList = append(osList, strings.ToLower(p.OSType()))
		archList = append(archList, strings.ToLower(p.Architecture()))
	}

	pluginData := map[string][]string{
		"name":    nameList,
		"version": versionList,
		"os":      osList,
		"arch":    archList,
	}
	pluginDataMarshal, err := json.Marshal(&pluginData)
	if err != nil {
		log.GetLogger().Error("pluginLocalListReport: Marshal err: ", err.Error())
		return
	}
	localArch, _ := GetArch()
	metrics.GetPluginLocalListEvent(
		"pluginList", string(pluginDataMarshal),
		"localArch", localArch,
		"localOsType", osutil.GetOsType(),
	).ReportEvent()
}

func parsePluginHealthCheck(content string) (PluginStatusResponse, error) {
	// 从check_update接口的响应数据中解析出插件升级信息
	pluginStatusResp := PluginStatusResponse{}
	if err := json.Unmarshal([]byte(content), &pluginStatusResp); err != nil {
		log.GetLogger().Errorf("parse pluginUpdateCheck fail: %s", content)
		return pluginStatusResp, err
	}
	return pluginStatusResp, nil
}

func parsePluginUpdateCheck(content string) (PluginUpdateCheckResponse, error) {
	// 从check_update接口的响应数据中解析出插件升级信息
	pluginUpdateCheckResp := PluginUpdateCheckResponse{}
	if err := json.Unmarshal([]byte(content), &pluginUpdateCheckResp); err != nil {
		log.GetLogger().Errorf("parse pluginUpdateCheck fail: %s", content)
		return pluginUpdateCheckResp, err
	}
	return pluginUpdateCheckResp, nil
}

func refreshTimer(timer *timermanager.Timer, nextInterval int) error {
	mutableSchedule, ok := timer.Schedule.(*timermanager.MutableScheduled)
	if !ok {
		return errors.New("Unexpected schedule type of heartbeat timer")
	}
	mutableSchedule.SetInterval(time.Duration(nextInterval) * time.Second)
	timer.RefreshTimer()
	return nil
}

func _findAllInstalledPlugins() ([]PluginInfo, error) {
	installedPlugins, err := LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}
	defer installedPlugins.Close()

	_, pluginInfoList, err := installedPlugins.FindAll()
	return pluginInfoList, err
}

func _truncateStatusRequestFields(request *PluginStatusResquest) *PluginStatusResquest {
	for i, _ := range request.Plugin {
		pluginStatus := &request.Plugin[i]

		// 太长的名称和版本号字段进行截断
		if len(pluginStatus.Name) > PLUGIN_NAME_MAXLEN {
			pluginStatus.Name = pluginStatus.Name[:PLUGIN_NAME_MAXLEN]
		}
		if len(pluginStatus.Version) > PLUGIN_VERSION_MAXLEN {
			pluginStatus.Version = pluginStatus.Version[:PLUGIN_VERSION_MAXLEN]
		}
	}

	return request
}
