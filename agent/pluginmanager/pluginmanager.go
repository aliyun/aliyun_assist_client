package pluginmanager

import (
	"bytes"
	"encoding/json"
	"errors"
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
	"github.com/aliyun/aliyun_assist_client/agent/taskengine/timermanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/jsonutil"
)

/*
健康检查：
1. 初始化一个定时任务：
	1. 读取installed_plugins文件获得插件信息，检查是否有插件需要上报
	2. 如果需要上报，执行 acs-plugin-manager --status 得到所有插件的状态 （acs-plugin-manager 提供status接口）
	3. 汇总后报告给服务端

插件升级：
1. 初始化一个定时任务：
	1. 读取installed_plugins文件获得所有插件信息，检查是否有已安装的插件
	2. 所有插件信息上报服务端
	3. 遍历服务端响应的升级列表
		1. 执行 `acs-plugin-manager --exec -P ecs_tool ----pluginVersion 1.0`。升级插件
*/

const (
	// DefaultPluginHealthIntervalSeconds 插件健康检查的间隔时间
	DefaultPluginHealthIntervalSeconds = 30 * 60
	// DefaultPluginUpdateCheckIntervalSeconds 插件检查升级的间隔时间
	DefaultPluginUpdateCheckIntervalSeconds = 30 * 60

)

var (
	_pluginHealthTimer          *timermanager.Timer
	_pluginHealthTimerInitLock  sync.Mutex
	pluginHealthIntervalSeconds = DefaultPluginHealthIntervalSeconds

	_pluginUpdateCheck               *timermanager.Timer
	_pluginUpdateCheckInitLock       sync.Mutex
	pluginUpdateCheckIntervalSeconds = DefaultPluginUpdateCheckIntervalSeconds
)

func loadIntervalConf() {
	// 读取配置文件，如果有就读取并设置pluginHealthIntervalSeconds， pluginUpdateCheckIntervalSeconds
	cpath, _ := os.Executable()
	dir, _ := filepath.Abs(filepath.Dir(cpath))
	intervalPath := path.Join(dir, "config", "PluginCheckInterval")
	content, err := ioutil.ReadFile(intervalPath)
	if err != nil {
		return
	}
	contentStr := string(content)
	/*
		pluginHealthIntervalSeconds=60,
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
	if interval < 60 {
		interval = 60
	}
	if interval > 2 * 60 * 60 {
		interval = 2 * 60 * 60
	}
	if params=="pluginHealthIntervalSeconds" {
		pluginHealthIntervalSeconds = interval
		log.GetLogger().Infof("pluginHealthIntervalSeconds is init as %d seconds", interval)
	} else if params=="pluginUpdateCheckIntervalSeconds" {
		pluginUpdateCheckIntervalSeconds = interval
		log.GetLogger().Infof("pluginUpdateCheckIntervalSeconds is init as %d seconds", interval)
	}
}

func InitPluginCheckTimer() bool {
	loadIntervalConf()
	succ := true
	if err := initPluginHealthTimer(); err != nil {
		succ = false
		log.GetLogger().Errorln("Failed to initialize plugin health timer: " + err.Error())
	} else {
		log.GetLogger().Infoln("Initialize plugin health timer success")
	}

	if err := initPluginUpdateCheckTimer(); err != nil {
		succ = false
		log.GetLogger().Errorln("Failed to initialize plugin update_check timer: " + err.Error())
	} else {
		log.GetLogger().Infoln("Initialize plugin update_check timer success")
	}
	return succ
}

func initPluginHealthTimer() error {
	if _pluginHealthTimer == nil {
		_pluginHealthTimerInitLock.Lock()
		defer _pluginHealthTimerInitLock.Unlock()
		if _pluginHealthTimer == nil {
			log.GetLogger().Infof("initialize plugin health timer with interval %d seconds", pluginHealthIntervalSeconds)
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(pluginHealthCheck, pluginHealthIntervalSeconds)
			if err != nil {
				log.GetLogger().WithError(err).Error("create plugin health timer failed")
				return err
			}
			_pluginHealthTimer = timer
			go func() {
				// shuffle timer
				mills := rand.Intn(pluginHealthIntervalSeconds * 1000)
				time.Sleep(time.Duration(mills) * time.Millisecond)
				log.GetLogger().Info("run plugin check health timer")
				_, err = _pluginHealthTimer.Run()
				if err != nil {
					log.GetLogger().WithError(err).Error("run plugin check health timer failed")
				}
			}()
			return nil
		}
		return errors.New("plugin check health timer has been initialized")
	}
	return errors.New("plugin check health timer has been initialized")
}

func initPluginUpdateCheckTimer() error {
	if _pluginUpdateCheck == nil {
		_pluginUpdateCheckInitLock.Lock()
		defer _pluginUpdateCheckInitLock.Unlock()
		if _pluginUpdateCheck == nil {
			log.GetLogger().Infof("initialize plugin update check timer with interval %d seconds", pluginUpdateCheckIntervalSeconds)
			timerManager := timermanager.GetTimerManager()
			timer, err := timerManager.CreateTimerInSeconds(pluginUpdateCheck, pluginUpdateCheckIntervalSeconds)
			if err != nil {
				log.GetLogger().WithError(err).Error("create plugin update check timer failed")
				return err
			}
			_pluginUpdateCheck = timer
			go func() {
				// shuffle timer
				mills := rand.Intn(pluginUpdateCheckIntervalSeconds * 1000)
				time.Sleep(time.Duration(mills) * time.Millisecond)
				log.GetLogger().Info("run plugin check update timer")
				_, err = _pluginUpdateCheck.Run()
				if err != nil {
					log.GetLogger().WithError(err).Error("run plugin check update timer failed")
				}
			}()
			return nil
		}
		return errors.New("plugin check update timer has been initialized")
	}
	return errors.New("plugin check update timer has been initialized")
}

func pluginHealthCheck() {
	log.GetLogger().Info("pluginHealthCheck start")
	// 1.检查插件列表，如果没有插件就不需要健康检查
	pluginInfoList, err := loadPlugins()
	if err != nil {
		log.GetLogger().WithError(err).Errorf("pluginHealthCheck fail: loadPlugins fail")
		return
	}
	if pluginInfoList==nil {
		log.GetLogger().Infof("pluginHealthCheck cancle: there is no plugin")
		return
	}

	// 2.将插件状态发送给服务端
	pluginStatusRequest := PluginStatusResquest {
		Os: pluginInfoList[0].OSType,
		Arch: pluginInfoList[0].Arch,
	}
	persistPluginCount := 0
	pluginInfoMap := make(map[string]*PluginInfo)
	for _, pluginInfo := range pluginInfoList {
		pluginInfoMap[pluginInfo.Name] = &pluginInfo
		if pluginInfo.PluginType == PluginOneTime {
			pluginStatus := PluginStatus{
				PluginID: pluginInfo.PluginID,
				Name: pluginInfo.Name,
				Status: ONCE_INSTALLED,
				Version: pluginInfo.Version,
				Uptime: 0,
			}
			pluginStatusRequest.Plugin = append(pluginStatusRequest.Plugin, pluginStatus)
		} else if pluginInfo.PluginType == PluginPersist {
			persistPluginCount += 1
		}
	}

	if persistPluginCount > 0 {
		// 调用plugin-manager模块的 status接口，批量获取常驻插件状态
		mixedOutput := bytes.Buffer{}
		cmd := "acs-plugin-manager"
		arguments := []string{"--status"}
		_, _, err = syncRunKillGroup("", cmd, arguments, &mixedOutput, &mixedOutput, 20)
		if err != nil {
			log.GetLogger().WithError(err).Errorf("pluginHealthCheck fail: cmd run err: %s, cmd[%s %s] output[%s]", err.Error(), cmd, strings.Join(arguments, " "), mixedOutput.String())
			return
		}
		content := mixedOutput.Bytes()
		pluginStatusList := []PluginStatus{}
		if err := json.Unmarshal(content, &pluginStatusList); err != nil {
			log.GetLogger().Errorf("json.Unmarshal pluginStatusList error: %s, content: %s", err.Error(), string(content))
		}
		if len(pluginStatusList) == 0 {
			log.GetLogger().Infof("pluginHealthCheck : there is no persist plugin, content[%s]", string(content))
		}

		for _, pluginInfo := range pluginStatusList {
			pluginStatus := PluginStatus{
				PluginID: pluginInfo.PluginID,
				Name: pluginInfo.Name,
				Version: pluginInfo.Version,
				Status: pluginInfo.Status,
			}
			pluginStatusRequest.Plugin = append(pluginStatusRequest.Plugin, pluginStatus)
			// 状态异常的插件调用--start尝试拉起
			// if pluginInfo.Status != PERSIST_RUNNING {
			// 	log.GetLogger().Warnf("plugin[%s] is not running, try to start it", pluginInfo.Name)
			// 	go func() {
			// 		command := "acs-plugin-manager"
			// 		arguments := []string{"-e", "--local", "-P", pluginInfo.Name, "-p", "--start"}
			// 		timeout := 60
			// 		if pluginInfoPtr, ok := pluginInfoMap[pluginInfo.Name]; ok && pluginInfoPtr.Timeout != "" {
			// 			if t, err := strconv.Atoi(pluginInfoPtr.Timeout); err == nil {
			// 				timeout = t
			// 			}
			// 		}
			// 		syncRunKillGroup("", command, arguments, nil, nil, timeout)
			// 	}()
			// }
		}
	}
	requestPayloadBytes, err := json.Marshal(pluginStatusRequest)
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheck fail: pluginStatusList marshal fail")
		return
	}
	requestPayload := string(requestPayloadBytes)
	url := util.GetPluginHealthService()
	_, err = util.HttpPost(url, requestPayload, "")

	for i := 0; i < 3 && err != nil; i++ {
		log.GetLogger().Infof("upload pluginStatusList fail, need retry: %s", requestPayload)
		time.Sleep(time.Duration(2) * time.Second)
		_, err = util.HttpPost(url, requestPayload, "")
	}
	if err != nil {
		log.GetLogger().WithError(err).Error("pluginHealthCheck fail: post pluginStatusList fail")
		return
	}
	log.GetLogger().Info("pluginHealthCheck success")
}

func pluginUpdateCheck() {
	log.GetLogger().Info("pluginUpdateCheck start")
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
		if pluginInfo.PluginType == PluginPersist {
			persistPluginCount += 1
		}
	}
	if persistPluginCount == 0 {
		log.GetLogger().Info("pluginUpdateCheck cancle: there is no persist plugin to check update")
		return
	}

	// 2.生成插件升级检查的请求数据 (只关注常驻插件）
	pluginUpdateCheckRequest := PluginUpdateCheckRequest{
		Os: pluginInfoList[0].OSType,
		Arch: pluginInfoList[0].Arch,
	}
	for _, pluginInfo := range pluginInfoList {
		if pluginInfo.PluginType == PluginPersist {
			pluginUpdateCheckRequest.Plugin = append(pluginUpdateCheckRequest.Plugin, PluginUpdateCheck{
				PluginID: pluginInfo.PluginID,
				Name: pluginInfo.Name,
				Version:  pluginInfo.Version,
			})
		} else {
			log.GetLogger().Infof("pluginUpdateCheck: pluginName[%s] pluginType[%d] PluginPersist[%d] not persist plugin", pluginInfo.Name, pluginInfo.PluginType, PluginPersist)
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
	_, err = parsePluginUpdateCheck(resp)
	if err != nil {
		log.GetLogger().WithError(err).Errorf("pluginUpdateCheck fail: parse pluginUpdateInfo from resp fail: %s", resp)
		return
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
	if len(installedPlugins.PluginList)==0 {
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
