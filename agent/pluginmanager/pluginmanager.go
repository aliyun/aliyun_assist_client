package pluginmanager

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/jsonutil"
)

/*
健康检查：两种并行的周期上报插件状态的方式
	1. 调用 acs-plugin-manager --status，通过常驻插件的 --status 接口获得插件状态
	2. 读取常驻插件目录下的 heartbeat 文件获得常驻插件的心跳时间戳，通过心跳时间戳判断插件状态
		通过heartbeat判断插件状态后可以进行懒上报，只在插件状态变化时进行上报；默认每次都上报，受服务端返回的reportType字段控制
		lastPluginStatusRecord 用来记录上一次各插件的状态


插件升级：
1. 初始化一个定时任务：
	1. 读取installed_plugins文件获得所有插件信息，检查是否有已安装的插件
	2. 所有插件信息上报服务端
	3. 遍历服务端响应的升级列表
		1. 执行 `acs-plugin-manager --exec -P ecs_tool ----pluginVersion 1.0`。升级插件
*/

var (
	pluginHealthScanInterval  = 15 * 60 // 通过常驻插件的 --status 接口获取插件状态的时间间隔
	pluginHealthPullInterval  = 5 * 60  // 通过读取常驻插件心跳时间戳判断插件状态的时间间隔
	avoidTime = 60 // 单位秒， pluginHealthPull 与 pluginHealthScan 的相距时间需要大于avoidTime，避免短时间内重复上报
	lastPluginHealthCheckTime int64     // 记录上一次插件状态检查的时间，避免两种插件状态上报方式同时上报
	pluginHealthCheckTimeMut  sync.Mutex
	lazyReport bool
	lastPluginStatusRecord = map[string]string{}

	// pluginUpdateCheckIntervalSeconds 插件检查升级的间隔时间
	pluginUpdateCheckIntervalSeconds = 15 * 60
)

var (
	pluginHealthScanTimer *time.Timer
	pluginHealthPullTimer *time.Timer

	pluginUpdateTimer *time.Timer
)

func loadIntervalConf() {
	// 读取配置文件，如果有就读取并设置相关变量。用来测试的
	cpath, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(cpath))
	intervalPath := path.Join(dir, "config", "PluginCheckInterval")
	content, err := ioutil.ReadFile(intervalPath)
	if err != nil {
		return
	}
	contentStr := string(content)
	/*
		pluginHealthScanInterval=60,
		pluginHealthPullInterval=60,
		pluginUpdateCheckIntervalSeconds=60
	*/
	pvsList := strings.Split(contentStr, ",")
	for _, pvs := range pvsList {
		pvs = strings.TrimSpace(pvs)
		if len(pvs) == 0 {
			continue
		}
		pv := strings.Split(pvs, "=")
		if len(pv) != 2 {
			continue
		}
		interval, err := strconv.Atoi(pv[1])
		if err != nil {
			continue
		}
		setInterval(pv[0], interval)
	}
}

func setInterval(params string, interval int) {
	// if interval < 60 {
	// 	interval = 60
	// }
	// if interval > 2*60*60 {
	// 	interval = 2 * 60 * 60
	// }
	if params == "pluginHealthScanInterval" {
		pluginHealthScanInterval = interval
		log.GetLogger().Infof("pluginHealthScanInterval is init as %d seconds", interval)
	} else if params == "pluginHealthPullInterval" {
		pluginHealthPullInterval = interval
		log.GetLogger().Infof("pluginHealthPullInterval is init as %d seconds", interval)
	} else if params == "pluginUpdateCheckIntervalSeconds" {
		pluginUpdateCheckIntervalSeconds = interval
		log.GetLogger().Infof("pluginUpdateCheckIntervalSeconds is init as %d seconds", interval)
	}
}

func InitPluginCheckTimer() {
	// health check
	go func() {
		randSleep := rand.Intn(60 * 1000)
		pluginHealthScanTimer = time.NewTimer(time.Duration(randSleep) * time.Millisecond)
		for {
			_ = <-pluginHealthScanTimer.C
			pluginHealthCheckScan()
		}
	}()
	go func() {
		randSleep := rand.Intn(60 * 1000)
		// 确保pluginHealthCheckScan先上报
		pluginHealthPullTimer = time.NewTimer(time.Duration(60000 + randSleep) * time.Millisecond)
		for {
			_ = <-pluginHealthPullTimer.C
			pluginHealthCheckPull()
		}
	}()
	// update check
	// go func() {
	// 	randSleep := rand.Intn(60 * 1000)
	// 	time.Sleep(time.Duration(randSleep) * time.Millisecond)
	// 	pluginUpdateTimer = time.NewTimer(time.Duration(pluginUpdateCheckIntervalSeconds) * time.Second)
	// 	pluginUpdateCheck()
	// 	for {
	// 		_ = <-pluginUpdateTimer.C
	// 		pluginUpdateCheck()
	// 	}
	// }()
}

func pluginHealthCheckScan() {
	pluginHealthCheckTimeMut.Lock()
	lastPluginHealthCheckTime = time.Now().Unix()
	pluginHealthCheckTimeMut.Unlock()

	log.GetLogger().Info("pluginHealthCheckScan: start")
	defer func() {
		pluginHealthScanTimer.Reset(time.Duration(pluginHealthScanInterval) * time.Second)
	}()
	// 1.检查插件列表，如果没有插件就不需要健康检查
	pluginInfoList, err := loadPlugins()
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheckScan: loadPlugins err: " + err.Error())
		return
	}
	if pluginInfoList == nil {
		log.GetLogger().Infof("pluginHealthCheckScan: there is no plugin")
		return
	}

	// 2.将插件状态发送给服务端
	pluginStatusRequest := PluginStatusResquest{
		Plugin: []PluginStatus{},
	}
	persistPluginCount := 0
	pluginInfoMap := make(map[string]*PluginInfo)
	for _, pluginInfo := range pluginInfoList {
		pluginInfoMap[pluginInfo.Name] = &pluginInfo
		if pluginInfo.PluginType() == PLUGIN_ONCE {
			pluginStatus := PluginStatus{
				Name:     pluginInfo.Name,
				Status:   ONCE_INSTALLED,
				Version:  pluginInfo.Version,
			}
			if pluginInfo.IsRemoved {
				pluginStatus.Status = REMOVED
			}
			// 太长的名称和版本号字段进行截断
			if len(pluginStatus.Name) > PLUGIN_NAME_MAXLEN {
				pluginStatus.Name = pluginStatus.Name[:PLUGIN_NAME_MAXLEN]
			}
			if len(pluginStatus.Version) > PLUGIN_VERSION_MAXLEN {
				pluginStatus.Version = pluginStatus.Version[:PLUGIN_VERSION_MAXLEN]
			}
			pluginStatusRequest.Plugin = append(pluginStatusRequest.Plugin, pluginStatus)
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
			log.GetLogger().Errorf("pluginHealthCheckScan: cmd run err: %s, cmd[%s %s] output[%s]", err.Error(), cmd, strings.Join(arguments, " "), mixedOutput.String())
			return
		}
		content := mixedOutput.Bytes()
		pluginStatusList := []PluginStatus{}
		if err := json.Unmarshal(content, &pluginStatusList); err != nil {
			log.GetLogger().Errorf("pluginHealthCheckScan: json.Unmarshal pluginStatusList error: %s, content: %s", err.Error(), string(content))
		}
		if len(pluginStatusList) == 0 {
			log.GetLogger().Infof("pluginHealthCheckScan: there is no persist plugin, content[%s]", string(content))
		}

		for _, pluginInfo := range pluginStatusList {
			pluginStatus := PluginStatus{
				Name:     pluginInfo.Name,
				Version:  pluginInfo.Version,
				Status:   pluginInfo.Status,
			}
			// 太长的名称和版本号字段进行截断
			if len(pluginStatus.Name) > PLUGIN_NAME_MAXLEN {
				pluginStatus.Name = pluginStatus.Name[:PLUGIN_NAME_MAXLEN]
			}
			if len(pluginStatus.Version) > PLUGIN_VERSION_MAXLEN {
				pluginStatus.Version = pluginStatus.Version[:PLUGIN_VERSION_MAXLEN]
			}
			if pluginInfo.Status != PERSIST_RUNNING && pluginInfo.Status != REMOVED {
				// // 状态异常的常驻插件本次不上报，acs-plugin-manager调用--start拉起后会单独上报该插件的状态
				log.GetLogger().Warnf("plugin[%s] is not running, try to start it", pluginInfo.Name)
				go func() {
					randSleep := rand.Intn(10 * 1000)
					time.Sleep(time.Duration(randSleep) * time.Millisecond)
					command := "acs-plugin-manager"
					arguments := []string{"-e", "--local", "-P", pluginInfo.Name, "-p", "--start"}
					timeout := 60
					if pluginInfoPtr, ok := pluginInfoMap[pluginInfo.Name]; ok && pluginInfoPtr.Timeout != "" {
						if t, err := strconv.Atoi(pluginInfoPtr.Timeout); err == nil {
							timeout = t
						}
					}
					syncRunKillGroup("", command, arguments, nil, nil, timeout)
				}()
			} else {
				// 状态正常的常驻插件进行上报
				pluginStatusRequest.Plugin = append(pluginStatusRequest.Plugin, pluginStatus)
			}
		}
	}
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
	if pluginStatusResp.PullInterval > 0 {
		pluginHealthPullInterval = pluginStatusResp.PullInterval
	}
	if pluginStatusResp.ScanInterval > 0 {
		pluginHealthScanInterval = pluginStatusResp.ScanInterval
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
	defer func() {
		pluginHealthPullTimer.Reset(time.Duration(pluginHealthPullInterval) * time.Second)
	}()

	now := time.Now().Unix()
	needWait := int64(avoidTime) - (now - lastTime)
	if needWait > 0 {
		log.GetLogger().Infof("pluginHealthCheckPull: last pluginHealthCheckScan started [%d] seconds ago, need wait [%d] second for avoidTime[%d]", now - lastTime, needWait, avoidTime)
		time.Sleep(time.Duration(needWait) * time.Second)
		now = time.Now().Unix()
	}
	remainTime := lastTime + int64(pluginHealthScanInterval) - now
	if remainTime < int64(avoidTime) {
		log.GetLogger().Infof("pluginHealthCheckPull: next pluginHealthCheckScan will start in [%d] seconds, less than avoidTime[%d], so cancel this pluginHealthCheckPull", remainTime, avoidTime)
		return
	}
	log.GetLogger().Info("pluginHealthCheckPull: start")
	// 1.检查插件列表，如果没有插件就不需要健康检查
	pluginInfoList, err := loadPlugins()
	if err != nil {
		log.GetLogger().Error("pluginHealthCheckPull: loadPlugins err: " + err.Error())
		return
	}
	if pluginInfoList == nil {
		log.GetLogger().Infof("pluginHealthCheckPull: there is no plugin")
		return
	}

	// 2.获取插件状态
	pluginStatusRequest := PluginStatusResquest{
		Plugin: []PluginStatus{},
	}
	pluginDir, err := getPluginPath()
	if err != nil {
		log.GetLogger().Error("pluginHealthCheckPull: getPluginPath err: ", err.Error())
		return
	}
	curPluginStatusRecord := map[string]string{}
	for _, pluginInfo := range pluginInfoList {
		if pluginInfo.PluginType() == PLUGIN_PERSIST && !pluginInfo.IsRemoved {
			// 常驻型插件且未被删除：检查并读取插件目录下的heartbeat文件
			heartbeatPath := filepath.Join(pluginDir, pluginInfo.Name, pluginInfo.Version, "heartbeat")
			if util.CheckFileIsExist(heartbeatPath) {
				content, err := ioutil.ReadFile(heartbeatPath)
				if err != nil {
					log.GetLogger().Errorf("pluginHealthCheckPull: Read heartbeat file err, heartbeat[%s], err: %s", heartbeatPath, err.Error())
					continue
				}
				timestampStr := strings.TrimSpace(string(content))
				timestamp, err := strconv.ParseInt(timestampStr, 10, 0)
				if err != nil {
					log.GetLogger().Errorf("pluginHealthCheckPull: Parse heartbeat file err, heartbeat[%s], content[%s] err: %s", heartbeatPath, timestampStr, err.Error())
					continue
				}
				status := PERSIST_RUNNING
				if now - timestamp > int64(pluginInfo.HeartbeatInterval + 5) {
					status = PERSIST_FAIL
				}
				curPluginStatusRecord[pluginInfo.Name] = status
				pluginStatus := PluginStatus{
					Name:     pluginInfo.Name,
					Status:   status,
					Version:  pluginInfo.Version,
				}
				if len(pluginStatus.Name) > PLUGIN_NAME_MAXLEN {
					pluginStatus.Name = pluginStatus.Name[:PLUGIN_NAME_MAXLEN]
				}
				if len(pluginStatus.Version) > PLUGIN_VERSION_MAXLEN {
					pluginStatus.Version = pluginStatus.Version[:PLUGIN_VERSION_MAXLEN]
				}
				pluginStatusRequest.Plugin = append(pluginStatusRequest.Plugin, pluginStatus)
			}
		}
	}
	if len(pluginStatusRequest.Plugin) == 0 {
		log.GetLogger().Infof("pluginHealthCheckPull: there is no persist plugin with heartbeat")
		return
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
	nextInterval := pluginUpdateCheckIntervalSeconds
	defer func() {
		pluginUpdateTimer.Reset(time.Duration(nextInterval) * time.Second)
	}()
	// 1.检查插件列表，如果没有插件就不需要升级检查
	pluginInfoList, err := loadPlugins()
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginUpdateCheck fail: loadPlugins fail")
		return
	}
	if pluginInfoList == nil {
		log.GetLogger().Info("pluginUpdateCheck cancle: there is no plugins")
		return
	}
	persistPluginCount := 0
	for _, pluginInfo := range pluginInfoList {
		if pluginInfo.PluginType() == PLUGIN_PERSIST {
			persistPluginCount += 1
		}
	}
	if persistPluginCount == 0 {
		log.GetLogger().Info("pluginUpdateCheck cancle: there is no persist plugin to check update")
		return
	}

	// 2.生成插件升级检查的请求数据 (只关注常驻插件）
	pluginUpdateCheckRequest := PluginUpdateCheckRequest{
		Os:   pluginInfoList[0].OSType,
		Arch: pluginInfoList[0].Arch,
	}
	for _, pluginInfo := range pluginInfoList {
		if pluginInfo.PluginType() == PLUGIN_PERSIST {
			pluginUpdateCheckRequest.Plugin = append(pluginUpdateCheckRequest.Plugin, PluginUpdateCheck{
				PluginID: pluginInfo.PluginID,
				Name:     pluginInfo.Name,
				Version:  pluginInfo.Version,
			})
		} else {
			log.GetLogger().Infof("pluginUpdateCheck: pluginName[%s] pluginType[%s]not persist plugin", pluginInfo.Name, pluginInfo.PluginType())
		}
	}

	requestPayloadBytes, err := json.Marshal(pluginUpdateCheckRequest)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginUpdateCheck fail: pluginStatusList marshal fail")
		return
	}
	requestPayload := string(requestPayloadBytes)
	log.GetLogger().Infof("pluginUpdateCheck requestPayload: %s", requestPayload)
	url := util.GetPluginUpdateCheckService()
	resp, err := util.HttpPost(url, requestPayload, "")

	for i := 0; i < 3 && err != nil; i++ {
		log.GetLogger().Infof("request updateCheck fail, need retry: %s", requestPayload)
		time.Sleep(time.Duration(2) * time.Second)
		resp, err = util.HttpPost(url, requestPayload, "")
	}
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginUpdateCheck fail: post pluginStatusList fail")
		return
	}

	// 3. 从check_update接口的响应数据中解析出要升级插件的信息
	var pluginUpdateCheckResp PluginUpdateCheckResponse
	pluginUpdateCheckResp, err = parsePluginUpdateCheck(resp)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("pluginUpdateCheck fail: parse pluginUpdateInfo from resp fail: %s", resp)
		return
	}
	if pluginUpdateCheckResp.NextInterval > 0 {
		nextInterval = pluginUpdateCheckResp.NextInterval
	}

	// 4.遍历pluginUpdateInfoList，对需要升级的插件执行升级操作
	// fileDir, err := getPluginPath()
	// if err != nil {
	// 	log.GetLogger().WithError(err).Error("pluginUpdateCheck fail: get pluginPath fail")
	// 	return
	// }
	// fileDir += string(os.PathSeparator)
	// pluginVersion := make(map[string]string)
	// for _, pluginInfo := range pluginInfoList {
	// 	pluginVersion[pluginInfo.Name] = pluginInfo.Version
	// }
	// log.GetLogger().Infof("the pluginUpdateCheckResponse.Plugin len is %d", len(pluginUpdateCheckResponse.Plugin))
	// for _, pluginUpdateInfo := range pluginUpdateCheckResponse.Plugin {
	// 	if pluginUpdateInfo.NeedUpdate != 1 {
	// 		continue
	// 	}
	// 	info := pluginUpdateInfo.Info
	// 	// 检查版本号 新的版本号<=原有的版本号 就跳过不安装
	// 	if version, ok := pluginVersion[info.Name]; ok && versionutil.CompareVersion(info.Version, version) <= 0 {
	// 		log.GetLogger().Infof("pluginName[%s] currentVersion[%s] updateVersion[%s] will not update", info.Name, pluginVersion[info.Name], info.Version)
	// 		continue
	// 	}
	// 	// 只打印要升级插件的日志，暂时不执行升级操作
	// 	if version, ok := pluginVersion[info.Name]; ok {
	// 		log.GetLogger().Infof("will upgrade plugin[%s] from version[%s] to versiont[%s]", pluginUpdateInfo.Info.Name, version, pluginUpdateInfo.Info.Version)
	// 	} else {
	// 		log.GetLogger().Infof("will upgrade plugin[%s] to versiont[%s]", pluginUpdateInfo.Info.Name, pluginUpdateInfo.Info.Version)
	// 	}
	// }
	log.GetLogger().Infof("pluginUpdateCheck success response: %s", resp)
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

func loadPlugins() ([]PluginInfo, error) {
	pluginPath, err := getPluginPath()
	if err != nil {
		return nil, err
	}
	pluginPath += string(os.PathSeparator) + "installed_plugins"
	if !util.CheckFileIsExist(pluginPath) {
		return nil, nil
	}
	installedPlugins := InstalledPlugins{}
	if err := jsonutil.UnmarshalFile(pluginPath, &installedPlugins); err != nil {
		if content, err1 := ioutil.ReadFile(pluginPath); err1 == nil {
			log.GetLogger().Errorf("Unmarshal installedPlugins fail content: %s", string(content))
		}
		return nil, err
	}
	if len(installedPlugins.PluginList) == 0 {
		return nil, nil
	}
	return installedPlugins.PluginList, nil
}

func getPluginPath() (string, error) {
	pluginDir, err := util.GetCurrentPath()
	if err != nil {
		return "", err
	}
	pluginDir += ".." + string(os.PathSeparator) + "plugin"
	util.MakeSurePath(pluginDir)
	return pluginDir, nil
}
