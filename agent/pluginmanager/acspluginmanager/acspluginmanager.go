package acspluginmanager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/shlex"
	"github.com/rodaine/table"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	. "github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/versionutil"
	"github.com/aliyun/aliyun_assist_client/common/fuzzyjson"
)

type pluginConfig struct {
	Name              string      `json:"name"`
	Arch              string      `json:"arch"`
	OsType            string      `json:"osType"`
	RunPath           string      `json:"runPath"`
	Timeout           string      `json:"timeout"`
	Publisher         string      `json:"publisher"`
	Version           string      `json:"version"`
	PluginType_       interface{} `json:"pluginType"`
	HeartbeatInterval int         `json:"heartbeatInterval"`
	pluginTypeStr     string
}

func (pc *pluginConfig) PluginType() string {
	if pc.pluginTypeStr == "" {
		switch pc.PluginType_.(type) {
		case string:
			pt, _ := pc.PluginType_.(string)
			if pt == PLUGIN_ONCE {
				pc.pluginTypeStr = PLUGIN_ONCE
			} else if pt == PLUGIN_PERSIST {
				pc.pluginTypeStr = PLUGIN_PERSIST
			} else {
				pc.pluginTypeStr = PLUGIN_UNKNOWN
			}
		case float64:
			pt, _ := pc.PluginType_.(float64)
			if pt == float64(PLUGIN_ONCE_INT) {
				pc.pluginTypeStr = PLUGIN_ONCE
			} else if pt == float64(PLUGIN_PERSIST_INT) {
				pc.pluginTypeStr = PLUGIN_PERSIST
			} else {
				pc.pluginTypeStr = PLUGIN_UNKNOWN
			}
		case nil:
			pc.pluginTypeStr = PLUGIN_ONCE
		default:
			pc.pluginTypeStr = PLUGIN_UNKNOWN
		}
	}
	return pc.pluginTypeStr
}

type PluginManager struct {
	Verbose bool
	Yes     bool
}

var PLUGINDIR string

const Separator = string(filepath.Separator)

func NewPluginManager(verbose bool) (*PluginManager, error) {
	var err error
	PLUGINDIR, err = util.GetPluginPath()
	if err != nil {
		return nil, err
	}

	return &PluginManager{
		Verbose: verbose,
		Yes:     true,
	}, nil
}

func printPluginInfo(pluginInfoList *[]PluginInfo) {
	tbl := table.New("Name", "Version", "Publisher", "OsType", "Arch", "PluginType")
	for i := 0; i < len(*pluginInfoList); i++ {
		pluginInfo := (*pluginInfoList)[i]
		// 已删除的插件不打印
		if pluginInfo.IsRemoved {
			continue
		}
		tbl.AddRow(pluginInfo.Name, pluginInfo.Version, pluginInfo.Publisher, pluginInfo.OSType, pluginInfo.Arch, pluginInfo.PluginType())
	}
	tbl.Print()
	fmt.Println()
}

// get pluginInfo by name from online
func getPackageInfo(pluginName, version string, withArch bool) ([]PluginInfo, error) {
	arch := ""
	if withArch {
		arch, _ = GetArch()
	}
	postValue := PluginListRequest{
		OsType:     "linux",
		PluginName: pluginName,
		Version:    version,
		Arch:       arch,
	}
	listRet := PluginListResponse{}
	if osutil.GetOsType() == osutil.OSWin {
		postValue.OsType = "windows"
	}
	postValueStr, err := fuzzyjson.Marshal(&postValue)
	if err != nil {
		return listRet.PluginList, err
	}
	// http 请求尝试3次
	log.GetLogger().Infof("Request /plugin/list, params[%s]", string(postValueStr))
	ret, err := util.HttpPost(util.GetPluginListService(), postValueStr, "json")
	if err != nil {
		retry := 2
		for retry > 0 && err != nil {
			retry--
			// pluginlist接口有流控，等一下再重试
			time.Sleep(time.Duration(3) * time.Second)
			ret, err = util.HttpPost(util.GetPluginListService(), postValueStr, "json")
		}
	}
	if err != nil {
		return listRet.PluginList, err
	}
	if err := fuzzyjson.Unmarshal(ret, &listRet); err != nil {
		return nil, err
	}
	return listRet.PluginList, nil
}

func getLocalPluginInfo(packageName, pluginVersion string) (*PluginInfo, error) {
	installedPlugins, err := LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}

	_, pluginInfo := installedPlugins.FindOneNotRemovedByNameAndOptionalVersion(packageName, pluginVersion)
	return pluginInfo, nil
}

func getOnlinePluginInfo(packageName, version string) (archMatch *PluginInfo, archNotMatch []string, err error) {
	// request all arch pluginInfos
	var pluginList []PluginInfo
	pluginList, err = getPackageInfo(packageName, version, false)
	if err != nil {
		return nil, nil, err
	}
	localArch, _ := GetArch()
	for idx, plugin := range pluginList {
		if plugin.Name == packageName {
			plugin.Arch = strings.ToLower(plugin.Arch)
			if plugin.Arch == "" || plugin.Arch == "all" || localArch == plugin.Arch {
				if archMatch == nil {
					archMatch = &pluginList[idx]
				// if plugin.Version > archMatch.Version, update archMatch
				} else if versionutil.CompareVersion(plugin.Version, archMatch.Version) > 0 {
					archMatch = &pluginList[idx]
				}
			} else {
				archNotMatch = append(archNotMatch, plugin.Arch)
			}
		}
	}
	return
}

func (pm *PluginManager) List(pluginName string, local bool) (exitCode int, err error) {
	var installedPlugins *InstalledPlugins
	var pluginInfoList []PluginInfo
	funcName := "List"
	exitCode = SUCCESS
	if local {
		installedPlugins, err = LoadInstalledPlugins()
		if err != nil {
			exitCode, _ = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
			return
		}
		if pluginName != "" {
			_, pluginInfoList = installedPlugins.FindManyByName(pluginName)
		} else {
			_, pluginInfoList = installedPlugins.FindAll()
		}
	} else {
		// just request pluginInfos with right arch
		pluginInfoList, err = getPackageInfo(pluginName, "", true)
		if err != nil {
			exitCode, _ = errProcess(funcName, GET_ONLINE_PACKAGE_INFO_ERR, err, "Get plugin info from online err: "+err.Error())
			return
		}
	}
	printPluginInfo(&pluginInfoList)
	return
}

// 打印常驻插件的状态，包括已删除的常驻插件
func (pm *PluginManager) ShowPluginStatus() (exitCode int, err error) {
	log.GetLogger().Infoln("Enter showPluginStatus")
	funcName := "ShowPluginStatus"
	exitCode = SUCCESS
	installedPlugins, err := LoadInstalledPlugins()
	if err != nil {
		exitCode, _ = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: " + err.Error())
		return
	}

	_, pluginList := installedPlugins.FindAll()
	log.GetLogger().Infof("Count of installed plugins: %d", len(pluginList))
	statusList := []PluginStatus{}
	pluginPath := PLUGINDIR + Separator
	paramList := []string{"--status"}
	for _, plugin := range pluginList {
		timeout := 60
		code := 0
		if t, err := strconv.ParseInt(plugin.Timeout, 10, 0); err == nil {
			timeout = int(t)
		}
		if plugin.PluginType() == PLUGIN_PERSIST {
			status := PluginStatus{
				Name:    plugin.Name,
				Version: plugin.Version,
				Status:  PERSIST_FAIL,
			}
			if plugin.IsRemoved {
				status.Status = REMOVED
			} else {
				pluginDir := filepath.Join(pluginPath, plugin.Name, plugin.Version)
				env := []string{
					"PLUGIN_DIR=" + pluginDir,
				}
				cmdPath := filepath.Join(pluginDir, plugin.RunPath)
				code, _, err = pm.executePlugin(cmdPath, paramList, timeout, env, true)
				if code == 0 && err == nil {
					status.Status = PERSIST_RUNNING
				}
				if err != nil {
					log.GetLogger().Errorf("ShowPluginStatus: executePlugin err, pluginName[%s] pluginVersion[%s]", plugin.Name, plugin.Version)
				}
			}
			statusList = append(statusList, status)
		}
	}
	content, err := fuzzyjson.Marshal(&statusList)
	if err != nil {
		log.GetLogger().Error("ShowPluginStatus err when marshal statusList, err: ", err.Error())
	}
	fmt.Println(content)
	return
}

func (pm *PluginManager) ExecutePlugin(file, pluginName, pluginId, params, separator, paramsV2, version string, local bool) (exitCode int, err error) {
	log.GetLogger().Infoln("Enter ExecutePlugin")
	if pm.Verbose {
		log.GetLogger().Infof("ExecutePlugin: file[%s], pluginName[%s], pluginId[%s], params[%s], separator[%s], paramsV2[%s], version[%s], local[%v]", file, pluginName, pluginId, params, separator, paramsV2, version, local)
	}
	var paramList []string
	timeout := 60
	if paramsV2 != "" {
		paramList, _ = shlex.Split(paramsV2)
	} else {
		if separator == "" {
			separator = ","
		}
		paramsSpace := strings.Replace(params, separator, " ", -1)
		paramList, _ = shlex.Split(paramsSpace)
	}
	if len(paramList) == 0 {
		paramList = nil
	}
	if file != "" {
		return pm.executePluginFromFile(file, paramList, timeout)
	}
	// execute plugin exe-file
	return pm.executePluginOnlineOrLocal(pluginName, pluginId, version, paramList, timeout, local)
}

// 根据插件名称删除插件，会删除该插件的整个目录（包括其中各版本的目录）
// 一次性插件：直接删除相应的目录并将installed_plugins中对应的插件标记为已删除（isRemoved=true）
// 常驻型插件：删除之前先调用插件的 --stop和 --uninstall，如果--uninstall退出码非0则不删除，否则像一次性插件一样删除目录并标记
func (pm *PluginManager) RemovePlugin(pluginName string) (exitCode int, err error) {
	defer func() {
		if exitCode != 0 || err != nil {
			fmt.Printf("RemovePlugin error, plugin[%s], err: %v\n", pluginName, err)
		} else {
			fmt.Printf("RemovePlugin success, plugin[%s]\n", pluginName)
		}
	}()
	const funcName = "RemovePlugin"

	installedPlugins, err := LoadInstalledPlugins()
	if err != nil {
		exitCode, _ = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
		return
	}

	idx, pluginInfo := installedPlugins.FindOneNotRemovedByName(pluginName)
	if pluginInfo == nil {
		exitCode, _ = errProcess(funcName, PACKAGE_NOT_FOUND, err, "plugin not exist "+pluginName)
		err = errors.New("Plugin " + pluginName + " not found in installed_plugins")
		return
	}

	if pluginInfo.PluginType() == PLUGIN_PERSIST {
		// 常驻型插件
		var (
			envPluginDir    string
			envPrePluginDir string
		)
		cmdPath := filepath.Join(PLUGINDIR, pluginInfo.Name, pluginInfo.Version, pluginInfo.RunPath)
		envPluginDir = filepath.Join(PLUGINDIR, pluginInfo.Name, pluginInfo.Version)

		var timeout int
		if timeout, err = strconv.Atoi(pluginInfo.Timeout); err != nil {
			timeout = 60
		}
		env := []string{
			"PLUGIN_DIR=" + envPluginDir,
			"PRE_PLUGIN_DIR=" + envPrePluginDir,
		}
		// --stop 停止插件进程
		paramList := []string{"--stop"}
		pm.executePlugin(cmdPath, paramList, timeout, env, false)
		// --uninstall 卸载插件服务
		paramList = []string{"--uninstall"}
		exitCode, _, err = pm.executePlugin(cmdPath, paramList, timeout, env, false)
		if exitCode != 0 || err != nil {
			return
		}
	}

	pluginInfo.IsRemoved = true // 标记为已删除
	// 更新installed_plugins文件
	installedPlugins.Update(idx, pluginInfo)
	if err = installedPlugins.Save(); err != nil {
		exitCode, _ = errProcess(funcName, DUMP_INSTALLEDPLUGINS_ERR, err, "Update installed_plugins file err: "+err.Error())
		return
	}
	if err = pm.ReportPluginStatus(pluginInfo.Name, pluginInfo.Version, REMOVED); err != nil {
		log.GetLogger().Errorf("Plugin[%s] is removed, but report the removed plugin to server error: %s", pluginInfo.Name, err.Error())
	}
	// 删除插件目录
	pluginDir := filepath.Join(PLUGINDIR, pluginInfo.Name)
	if err = os.RemoveAll(pluginDir); err != nil {
		exitCode, _ = errProcess(funcName, REMOVE_FILE_ERR, err, fmt.Sprintf("Remove plugin directory err, pluginDir[%s], err: %s", pluginDir, err.Error()))
		return
	}

	return
}

// run plugin from plugin_file.zip
func (pm *PluginManager) executePluginFromFile(file string, paramList []string, timeout int) (exitCode int, err error) {
	log.GetLogger().Infof("Enter executePluginFromFile")
	var (
		// variables for metrics
		pluginName    string
		pluginVersion string
		resource      string = "LocalFile"
		errorCode     string
		localArch     string
		pluginType    string

		cmdPath  string
		funcName string = "ExecutePluginFromFile"
		// 执行插件时要注入的环境变量
		envPluginDir    string // 当前执行的插件的执行目录
		envPrePluginDir string // 如果已有同名的其他版本插件，表示原有同名插件的执行目录；否则为空
	)
	defer func() {
		metrics.GetPluginExecuteEvent(
			"pluginName", pluginName,
			"pluginVersion", pluginVersion,
			"pluginType", pluginType,
			"resource", resource, // 插件来源，文件、本地已安装的、线上拉取的
			"exitCode", fmt.Sprint(exitCode),
			"errorCode", errorCode, // 错误码，plugin-manager定义的错误码，例如 PACKAGE_NOT_FOUND，如果这个字段非空表示插件没有正确执行，exitCode是plugin-manager定义的退出码；否则表示插件被正确调用，exitCode是插件执行的退出码
			"localArch", localArch,
			"localOsType", osutil.GetOsType(),
		).ReportEventSync()
	}()
	if pm.Verbose {
		fmt.Println("Execute plugin from file: ", file)
	}
	localArch, _ = GetArch()
	exitCode = SUCCESS
	if !util.CheckFileIsExist(file) {
		err = errors.New("File not exist: " + file)
		exitCode, errorCode = errProcess(funcName, PACKAGE_NOT_FOUND, err, "Package file not exist: "+file)
		return
	}
	idx := strings.LastIndex(file, Separator)
	pluginName = file
	dirName := "."
	if idx > 0 {
		pluginName = file[idx+1:]
		dirName = file[:idx]
	}
	idx = strings.Index(pluginName, ".zip")
	if idx <= 0 {
		err = errors.New("Package file isn`t a zip file: " + file)
		exitCode, errorCode = errProcess(funcName, PACKAGE_FORMAT_ERR, err, "Package file isn`t a zip file: "+file)
		return
	}
	pluginName = pluginName[:idx]
	dirName = filepath.Join(dirName, pluginName)
	util.MakeSurePath(dirName)
	if pm.Verbose {
		fmt.Printf("Unzip to %s ...\n", dirName)
	}
	unzipdir := dirName
	if err = util.Unzip(file, unzipdir); err != nil {
		exitCode, errorCode = errProcess(funcName, UNZIP_ERR, err, fmt.Sprintf("Unzip err, file is [%s], target dir is [%s], err is [%s]", file, unzipdir, err.Error()))
		return
	}
	config_path := filepath.Join(dirName, "config.json")
	if !util.CheckFileIsExist(config_path) {
		log.GetLogger().Errorf("File config.json not exist, %s.", config_path)
		config_path = filepath.Join(dirName, pluginName, "config.json")
		if !util.CheckFileIsExist(config_path) {
			log.GetLogger().Errorf("File config.json not exist, %s.", config_path)
			err = errors.New(fmt.Sprintf("File config.json not exist, %s.", config_path))
			exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("File config.json not exist, %s.", config_path))
			return
		}
		dirName = filepath.Join(dirName, pluginName)
	}
	config := pluginConfig{}
	var content []byte
	if content, err = fuzzyjson.UnmarshalFile(config_path, &config); err != nil {
		exitCode, errorCode = errProcess(funcName, UNMARSHAL_ERR, err, fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", string(content), err.Error()))
		return
	}
	pluginVersion = config.Version
	// 检查系统类型和架构是否符合
	if config.OsType != "" && strings.ToLower(config.OsType) != osutil.GetOsType() {
		err = errors.New("Plugin ostype not suit for this system")
		exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin ostype[%s] not suit for this system[%s]", config.OsType, osutil.GetOsType()))
		return
	}
	if config.Arch != "" && strings.ToLower(config.Arch) != "all" && strings.ToLower(config.Arch) != localArch {
		err = errors.New("Plugin arch not suit for this system")
		exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin arch[%s] not suit for this system[%s]", config.Arch, localArch))
		return
	}

	installedPlugins, err := LoadInstalledPlugins()
	if err != nil {
		exitCode, errorCode = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
		return
	}

	pluginIndex, plugin := installedPlugins.FindOneByName(config.Name)
	if plugin != nil && plugin.IsRemoved {
		// 之前的同名插件已经被删除，相当于重新安装
		installedPlugins.DeleteByKey(pluginIndex)
		pluginIndex = -1
		plugin = nil
	}
	if plugin != nil {
		envPrePluginDir = filepath.Join(PLUGINDIR, plugin.Name, plugin.Version)
		// has installed, check version
		if versionutil.CompareVersion(config.Version, plugin.Version) <= 0 {
			if !pm.Yes {
				yn := ""
				for {
					fmt.Printf("[%s %s] has installed, this package version[%s] is not newer, still install ? [y/n]: \n", plugin.Name, plugin.Version, config.Version)
					fmt.Scanln(&yn)
					if yn == "y" || yn == "n" {
						break
					}
				}
				if yn == "n" {
					log.GetLogger().Infoln("Execute plugin cancel...")
					fmt.Println("Execute plugin cancel...")
					return
				}
			}
			fmt.Printf("[%s %s] has installed, this package version[%s] is not newer, still install...\n", plugin.Name, plugin.Version, config.Version)
		} else {
			fmt.Printf("[%s %s] has installed, this package version[%s] is newer, keep install...\n", plugin.Name, plugin.Version, config.Version)
		}
	}
	if pluginIndex == -1 {
		plugin = &PluginInfo{
			Timeout: "60",
		}
	}
	if t, err := strconv.Atoi(config.Timeout); err != nil {
		config.Timeout = plugin.Timeout
	} else {
		timeout = t
	}
	plugin.Name = config.Name
	plugin.Arch = config.Arch
	plugin.OSType = config.OsType
	plugin.RunPath = config.RunPath
	plugin.Timeout = config.Timeout
	plugin.Publisher = config.Publisher
	plugin.Version = config.Version
	plugin.SetPluginType(config.PluginType())
	plugin.Url = "local"
	pluginType = plugin.PluginType()
	if config.HeartbeatInterval <= 0 {
		plugin.HeartbeatInterval = 60
	} else {
		plugin.HeartbeatInterval = config.HeartbeatInterval
	}
	var md5Str string
	md5Str, err = util.ComputeMd5(file)
	if err != nil {
		exitCode, errorCode = errProcess(funcName, MD5_CHECK_FAIL, err, "Compute md5 of plugin file err: "+err.Error())
		return
	}
	plugin.Md5 = md5Str

	pluginPath := filepath.Join(PLUGINDIR, plugin.Name, plugin.Version)
	envPluginDir = pluginPath
	util.MakeSurePath(pluginPath)
	util.CopyDir(dirName, pluginPath)
	cmdPath = filepath.Join(pluginPath, config.RunPath)
	if !util.CheckFileIsExist(cmdPath) {
		log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
		err = errors.New("Cmd file not exist: " + cmdPath)
		exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
		return
	}
	if osutil.GetOsType() != osutil.OSWin {
		if err = exec.Command("chmod", "744", cmdPath).Run(); err != nil {
			exitCode, errorCode = errProcess(funcName, EXECUTABLE_PERMISSION_ERR, err, "Make plugin file executable err: "+err.Error())
			return
		}
	}
	if pluginIndex == -1 {
		plugin.PluginID = "local_" + plugin.Name + "_" + plugin.Version
		installedPlugins.Insert(plugin)
	} else {
		installedPlugins.Update(pluginIndex, plugin)
	}
	if err = installedPlugins.Save(); err != nil {
		exitCode, errorCode = errProcess(funcName, DUMP_INSTALLEDPLUGINS_ERR, err, "Update installed_plugins file err: "+err.Error())
		return
	}
	fmt.Printf("Plugin[%s] installed!\n", plugin.Name)
	os.RemoveAll(unzipdir)

	env := []string{
		"PLUGIN_DIR=" + envPluginDir,
		"PRE_PLUGIN_DIR=" + envPrePluginDir,
	}
	exitCode, errorCode, err = pm.executePlugin(cmdPath, paramList, timeout, env, false)
	// 如果是常驻插件，且调用的接口有可能改变插件状态，需要主动上报一次插件状态
	if plugin.PluginType() == PLUGIN_PERSIST && needReportStatus(paramList) {
		status, err := pm.CheckAndReportPlugin(plugin.Name, plugin.Version, cmdPath, timeout, env)
		log.GetLogger().Infof("CheckAndReportPlugin : pluginName[%s] pluginVersion[%s] cmdPath[%s] timeout[%d] env[%v] status[%s], err: %v", plugin.Name, plugin.Version, cmdPath, timeout, env, status, err)
	}
	return
}

func (pm *PluginManager) executePluginOnlineOrLocal(pluginName string, pluginId string, pluginVersion string, paramList []string, timeout int, local bool) (exitCode int, err error) {
	log.GetLogger().Info("Enter executePluginOnlineOrLocal")
	var (
		// variables for metrics
		resource  string
		errorCode string
		localArch string

		useLocal   bool
		cmdPath    string
		pluginType string = PLUGIN_ONCE
		funcName   string = "ExecutePluginOnlineOrLocal"
		// 执行插件时要注入的环境变量
		envPluginDir    string // 当前执行的插件的执行目录
		envPrePluginDir string // 如果已有同名插件，表示已有同名插件的执行目录；否则为空
	)
	defer func() {
		if useLocal {
			resource = "LocalInstalled"
		} else {
			resource = "Online"
		}
		metrics.GetPluginExecuteEvent(
			"pluginName", pluginName,
			"pluginVersion", pluginVersion,
			"pluginType", pluginType,
			"resource", resource, // 插件来源，文件、本地已安装的、线上拉取的
			"exitCode", fmt.Sprint(exitCode),
			"errorCode", errorCode, // 错误码，plugin-manager定义的错误码，例如 PACKAGE_NOT_FOUND，如果这个字段非空表示插件没有正确执行，exitCode是plugin-manager定义的退出码；否则表示插件被正确调用，exitCode是插件执行的退出码
			"localArch", localArch,
			"localOsType", osutil.GetOsType(),
		).ReportEventSync()
	}()
	localArch, _ = GetArch()
	if !local {
		// didn't set --local, so local & online both try
		var localInfo *PluginInfo = nil
		var onlineInfo *PluginInfo = nil
		var onlineOtherArch []string
		localInfo, err = getLocalPluginInfo(pluginName, pluginVersion)
		if err != nil {
			exitCode, errorCode = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
			return
		}
		onlineInfo, onlineOtherArch, err = getOnlinePluginInfo(pluginName, pluginVersion)
		if err != nil {
			exitCode, errorCode = errProcess(funcName, GET_ONLINE_PACKAGE_INFO_ERR, err, "Get plugin info from online err: "+err.Error())
			return
		}
		if localInfo != nil {
			if onlineInfo != nil {
				// 本地和线上版本一致，使用本地插件文件
				if versionutil.CompareVersion(localInfo.Version, onlineInfo.Version) == 0 {
					log.GetLogger().Infof("ExecutePluginOnlineOrLocal: Plugin[%s], local version[%s] same to online version[%s], so use local package", pluginName, localInfo.Version, onlineInfo.Version)
					useLocal = true
				} else {
					// 本地和线上版本不一致，使用线上版本
					log.GetLogger().Infof("ExecutePluginOnlineOrLocal: Plugin[%s], local version[%s] different from online version[%s], so use online package", pluginName, localInfo.Version, onlineInfo.Version)
				}
			} else {
				useLocal = true
			}
		} else {
			if onlineInfo == nil {
				var tip string
				if len(onlineOtherArch) == 0 {
					tip = fmt.Sprintf("Could not found both local and online, package[%s] version[%s]\n", pluginName, pluginVersion)
				} else {
					localArch, _ = GetArch()
					tip = fmt.Sprintf("Could not found local package[%s] version[%s], found online package but it`s arch[%s] not match local_arch[%s] \n", pluginName, pluginVersion, strings.Join(onlineOtherArch, ", "), localArch)
				}
				err = errors.New("Could not found package")
				exitCode, errorCode = errProcess(funcName, PACKAGE_NOT_FOUND, err, tip)
				return
			}
		}
		// use local package
		if useLocal {
			if t, err := strconv.Atoi(localInfo.Timeout); err == nil {
				timeout = t
			}
			pluginPath := filepath.Join(PLUGINDIR, localInfo.Name, localInfo.Version)
			envPluginDir = pluginPath
			pluginName = localInfo.Name
			pluginVersion = localInfo.Version
			pluginType = localInfo.PluginType()
			cmdPath = filepath.Join(pluginPath, localInfo.RunPath)
		} else {
			// pull package
			filePath := filepath.Join(PLUGINDIR, pluginName+".zip")
			log.GetLogger().Infof("Downloading package from [%s], save to [%s] ", onlineInfo.Url, filePath)
			if err = util.HttpDownlod(onlineInfo.Url, filePath); err != nil {
				retry := 2
				for retry > 0 && err != nil {
					retry--
					time.Sleep(time.Second * 3)
					err = util.HttpDownlod(onlineInfo.Url, filePath)
				}
				if err != nil {
					exitCode, errorCode = errProcess(funcName, DOWNLOAD_FAIL, err, fmt.Sprintf("Downloading package failed, plugin.Url is [%s], err is [%s]", onlineInfo.Url, err.Error()))
					return
				}
			}
			log.GetLogger().Infoln("Check MD5...")
			md5Str := ""
			md5Str, err = util.ComputeMd5(filePath)
			if err != nil {
				exitCode, errorCode = errProcess(funcName, MD5_CHECK_FAIL, err, fmt.Sprintf("Compute md5 of plugin file[%s] err, plugin.Url is [%s], err is [%s]", filePath, onlineInfo.Url, err.Error()))
				return
			}
			if strings.ToLower(md5Str) != strings.ToLower(onlineInfo.Md5) {
				log.GetLogger().Errorf("Md5 not match, onlineInfo.Md5[%s], package file md5[%s]\n", onlineInfo.Md5, md5Str)
				err = errors.New("Md5 not macth")
				exitCode, errorCode = errProcess(funcName, MD5_CHECK_FAIL, err, fmt.Sprintf("Md5 not match, onlineInfo.Md5 is [%s], real md5 is [%s], plugin.Url is [%s]", onlineInfo.Md5, md5Str, onlineInfo.Url))
				return
			}
			unzipdir := filepath.Join(PLUGINDIR, onlineInfo.Name, onlineInfo.Version)
			util.MakeSurePath(unzipdir)
			log.GetLogger().Infoln("Unzip package...")
			if err = util.Unzip(filePath, unzipdir); err != nil {
				exitCode, errorCode = errProcess(funcName, UNZIP_ERR, err, fmt.Sprintf("Unzip package err, plugin.Url is [%s], err is [%s]", onlineInfo.Url, err.Error()))
				return
			}
			os.RemoveAll(filePath)
			config_path := filepath.Join(unzipdir, "config.json")
			if !util.CheckFileIsExist(config_path) {
				err = errors.New(fmt.Sprintf("File config.json not exist, %s.", config_path))
				exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("File config.json not exist, %s.", config_path))
				return
			}
			config := pluginConfig{}
			var content []byte
			if content, err = fuzzyjson.UnmarshalFile(config_path, &config); err != nil {
				exitCode, errorCode = errProcess(funcName, UNMARSHAL_ERR, err, fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", string(content), err.Error()))
				return
			}
			if config.HeartbeatInterval <= 0 {
				config.HeartbeatInterval = 60
			}
			if config.PluginType() != onlineInfo.PluginType() {
				tip := fmt.Sprintf("config.PluginType[%s] not match to pluginType[%s]", config.PluginType(), onlineInfo.PluginType())
				err = errors.New(tip)
				exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, tip)
				return
			}
			// 接口返回的插件信息中没有HeartbeatInterval字段，需要以插件包中的config.json为准
			onlineInfo.HeartbeatInterval = config.HeartbeatInterval
			onlineInfo.SetPluginType(config.PluginType())
			envPluginDir = filepath.Join(PLUGINDIR, config.Name, config.Version)
			pluginName = config.Name
			pluginVersion = config.Version
			pluginType = config.PluginType()
			if t, err := strconv.Atoi(config.Timeout); err == nil {
				timeout = t
			}
			cmdPath = filepath.Join(unzipdir, config.RunPath)
			// 检查系统类型和架构是否符合
			if strings.ToLower(onlineInfo.OSType) != "both" && strings.ToLower(onlineInfo.OSType) != osutil.GetOsType() {
				err = errors.New("Plugin ostype not suit for this system")
				exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin ostype[%s] not suit for this system[%s], plugin.Url is [%s]", onlineInfo.OSType, osutil.GetOsType(), onlineInfo.Url))
				return
			}
			if strings.ToLower(onlineInfo.Arch) != "all" && strings.ToLower(onlineInfo.Arch) != localArch {
				err = errors.New("Plugin arch not suit for this system")
				exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin arch[%s] not suit for this system[%s], plugin.Url is [%s]", onlineInfo.Arch, localArch, onlineInfo.Url))
				return
			}
			if !util.CheckFileIsExist(cmdPath) {
				log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
				err = errors.New("Cmd file not exist: " + cmdPath)
				exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
				return
			}
			if osutil.GetOsType() != osutil.OSWin {
				if err = os.Chmod(cmdPath, os.FileMode(0o744)); err != nil {
					exitCode, errorCode = errProcess(funcName, EXECUTABLE_PERMISSION_ERR, err, fmt.Sprintf("Make plugin file executable err, plugin.Url is [%s], err is [%s]", onlineInfo.Url, err.Error()))
					return
				}
			}

			// update INSTALLEDPLUGINS file
			var installedPlugins *InstalledPlugins
			installedPlugins, err = LoadInstalledPlugins()
			if err != nil {
				exitCode, errorCode = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
				return
			}

			pluginIndex, pluginInfo := installedPlugins.FindOneByName(onlineInfo.Name)
			if pluginIndex != -1 && pluginInfo.IsRemoved {
				// Actually from the database remove the record of removed plugin
				installedPlugins.DeleteByKey(pluginIndex)
				pluginIndex = -1
			}
			if pluginIndex == -1 {
				installedPlugins.Insert(onlineInfo)
			} else {
				envPrePluginDir = filepath.Join(PLUGINDIR, pluginInfo.Name, pluginInfo.Version)
				installedPlugins.Update(pluginIndex, onlineInfo)
			}
			err = installedPlugins.Save()
			if err != nil {
				exitCode, errorCode = errProcess(funcName, DUMP_INSTALLEDPLUGINS_ERR, err, "Update installed_plugins file err: "+err.Error())
				return
			}
		}
	} else {
		// execute local plugin
		useLocal = true
		var localInfo *PluginInfo
		localInfo, err = getLocalPluginInfo(pluginName, pluginVersion)
		if err != nil {
			exitCode, errorCode = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
			return
		} else if localInfo == nil {
			exitCode, errorCode = errProcess(funcName, PACKAGE_NOT_FOUND, err, fmt.Sprintf("Could not found local package [%s]", pluginName))
			return
		}
		envPluginDir = filepath.Join(PLUGINDIR, localInfo.Name, localInfo.Version)
		pluginName = localInfo.Name
		pluginVersion = localInfo.Version
		pluginType = localInfo.PluginType()
		if t, err := strconv.Atoi(localInfo.Timeout); err == nil {
			timeout = t
		}
		cmdPath = filepath.Join(envPluginDir, localInfo.RunPath)
	}

	env := []string{
		"PLUGIN_DIR=" + envPluginDir,
		"PRE_PLUGIN_DIR=" + envPrePluginDir,
	}
	exitCode, errorCode, err = pm.executePlugin(cmdPath, paramList, timeout, env, false)
	// 如果是常驻插件，且调用的接口有可能改变插件状态，需要主动上报一次插件状态
	if pluginType == PLUGIN_PERSIST && needReportStatus(paramList) {
		status, err := pm.CheckAndReportPlugin(pluginName, pluginVersion, cmdPath, timeout, env)
		log.GetLogger().Infof("CheckAndReportPlugin : pluginName[%s] pluginVersion[%s] cmdPath[%s] timeout[%d] env[%s] status[%s], err: %v", pluginName, pluginVersion, cmdPath, timeout, strings.Join(env, ","), status, err)
	}
	return
}

func (pm *PluginManager) executePlugin(cmdPath string, paramList []string, timeout int, env []string, quiet bool) (exitCode int, errorCode string, err error) {
	log.GetLogger().Infof("Enter executePlugin, cmdPath[%s] paramList[%v] paramCount[%d] timeout[%d]\n", cmdPath, paramList, len(paramList), timeout)
	funcName := "ExecutePlugin"
	if !util.CheckFileIsExist(cmdPath) {
		log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
		err = errors.New("Cmd file not exist: " + cmdPath)
		exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
		return
	}
	if pm.Verbose {
		fmt.Printf("Run cmd: %s, params: %v\n", cmdPath, paramList)
	}

	processCmd := process.NewProcessCmd()
	// set environment variable
	if env != nil && len(env) > 0 {
		processCmd.SetEnv(env)
	}
	status := process.Success
	commandName := cmdPath
	if filepath.Ext(cmdPath) == ".ps1" {
		commandName = "powershell"
		paramList = append([]string{cmdPath}, paramList...)
	}
	if quiet {
		exitCode, status, err = processCmd.SyncRun("", commandName, paramList, nil, nil, os.Stdin, nil, timeout)
	} else {
		exitCode, status, err = processCmd.SyncRun("", commandName, paramList, os.Stdout, os.Stderr, os.Stdin, nil, timeout)
	}
	if status == process.Fail {
		exitCode = EXECUTE_FAILED
	} else if status == process.Timeout {
		exitCode = EXECUTE_TIMEOUT
	}
	if !quiet {
		switch exitCode {
		case EXECUTE_FAILED:
			_, errorCode = errProcess(funcName, EXECUTE_FAILED, err, fmt.Sprintf("Execute plugin failed, err: %v", err))
		case EXECUTE_TIMEOUT:
			_, errorCode = errProcess(funcName, EXECUTE_TIMEOUT, err, fmt.Sprintf("Execute plugin timeout, timeout[%d] err: %v", timeout, err))
		}
	}
	log.GetLogger().Info(fmt.Sprintf("executePlugin: commandName: %s, params: %+q, exitCode: %d, timeout: %d, env: %v, err: %v\n", commandName, paramList, exitCode, timeout, env, err))
	return
}

func (pm *PluginManager) VerifyPlugin(url, params, separator, paramsV2 string) (exitCode int, err error) {
	log.GetLogger().Infof("Enter VerufyPlugin url[%s] params[%s] separator[%s]\n", url, params, separator)
	funcName := "VerifyPlugin"
	var paramList []string
	timeout := 60
	cmdPath := ""
	var (
		// 执行插件时要注入的环境变量
		envPluginDir    string // 当前执行的插件的执行目录
		envPrePluginDir string // 如果已有同名插件，表示已有同名插件的执行目录；否则为空
	)
	localArch, _ := GetArch()
	if paramsV2 != "" {
		paramList, _ = shlex.Split(paramsV2)
	} else {
		if separator == "" {
			separator = ","
		}
		paramsSpace := strings.Replace(params, separator, " ", -1)
		paramList, _ = shlex.Split(paramsSpace)
	}
	if len(paramList) == 0 {
		paramList = nil
	}

	// pull package
	fileName := url[strings.LastIndex(url, "/")+1:]
	filePath := PLUGINDIR + Separator + fileName
	log.GetLogger().Infoln("Downloading package from ", url)
	if len(url) > 4 && url[:4] == "http" {
		if err = util.HttpDownlod(url, filePath); err != nil {
			exitCode, _ = errProcess(funcName, DOWNLOAD_FAIL, err, fmt.Sprintf("Downloading package failed, url is [%s], err is [%s]", url, err.Error()))
			return
		}
	} else {
		if err = FileProtocolDownload(url, filePath); err != nil {
			exitCode, _ = errProcess(funcName, DOWNLOAD_FAIL, err, fmt.Sprintf("Downloading package failed, url is [%s], err is [%s]", url, err.Error()))
			return
		}
	}

	unzipdir := filepath.Join(PLUGINDIR, "verify_plugin_test")
	util.MakeSurePath(unzipdir)
	log.GetLogger().Infoln("Unzip package...")
	if err = util.Unzip(filePath, unzipdir); err != nil {
		exitCode, _ = errProcess(funcName, UNZIP_ERR, err, fmt.Sprintf("Unzip package err, url is [%s], err is [%s]", url, err.Error()))
		return
	}
	os.RemoveAll(filePath)

	configPath := filepath.Join(unzipdir, "config.json")
	if !util.CheckFileIsExist(configPath) {
		err = errors.New("Can not find the config.json")
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("File config.json not exist, %s.", configPath))
		return
	}
	config := pluginConfig{}
	var content []byte
	if content, err = fuzzyjson.UnmarshalFile(configPath, &config); err != nil {
		exitCode, _ = errProcess(funcName, UNMARSHAL_ERR, err, fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", string(content), err.Error()))
		return
	}
	// 检查系统类型和架构是否符合
	if config.OsType != "" && strings.ToLower(config.OsType) != "both" && strings.ToLower(config.OsType) != osutil.GetOsType() {
		err = errors.New("Plugin ostype not suit for this system")
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin ostype[%s] not suit for this system[%s], url is [%s]", config.OsType, osutil.GetOsType(), url))
		return
	}
	if config.Arch != "" && strings.ToLower(config.Arch) != "all" && strings.ToLower(config.Arch) != localArch {
		err = errors.New("Plugin arch not suit for this system")
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin arch[%s] not suit for this system[%s], url is [%s]", config.Arch, localArch, url))
		return
	}

	runPath := config.RunPath
	timeoutStr := config.Timeout
	envPluginDir = unzipdir
	cmdPath = filepath.Join(unzipdir, runPath)
	if !util.CheckFileIsExist(cmdPath) {
		err = errors.New("Can not find the cmd file")
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
		return
	}
	if osutil.GetOsType() != osutil.OSWin {
		err = exec.Command("chmod", "744", cmdPath).Run()
		if err != nil {
			exitCode, _ = errProcess(funcName, EXECUTABLE_PERMISSION_ERR, err, "Make plugin file executable err: "+err.Error())
			return
		}
	}
	timeout = 60
	if t, err := strconv.Atoi(timeoutStr); err != nil {
		fmt.Println("config.Timeout is invalid: ", config.Timeout)
	} else {
		timeout = t
	}

	env := []string{
		"PLUGIN_DIR=" + envPluginDir,
		"PRE_PLUGIN_DIR=" + envPrePluginDir,
	}
	exitCode, _, err = pm.executePlugin(cmdPath, paramList, timeout, env, false)
	return
}

// 向服务端上报某个插件状态
func (pm *PluginManager) ReportPluginStatus(pluginName, pluginVersion, status string) error {
	if len(pluginName) > PLUGIN_NAME_MAXLEN {
		pluginName = pluginName[:PLUGIN_NAME_MAXLEN]
	}
	if len(pluginVersion) > PLUGIN_VERSION_MAXLEN {
		pluginVersion = pluginVersion[:PLUGIN_VERSION_MAXLEN]
	}
	pluginStatusRequest := PluginStatusResquest{
		Plugin: []PluginStatus{
			{
				Name:    pluginName,
				Version: pluginVersion,
				Status:  status,
			},
		},
	}
	requestPayloadBytes, err := fuzzyjson.Marshal(pluginStatusRequest)
	if err != nil {
		log.GetLogger().WithError(err).Error("ReportPluginStatus: pluginStatusList marshal err: " + err.Error())
		return err
	}
	requestPayload := string(requestPayloadBytes)
	url := util.GetPluginHealthService()
	_, err = util.HttpPost(url, requestPayload, "")

	for i := 0; i < 3 && err != nil; i++ {
		log.GetLogger().Infof("ReportPluginStatus: upload pluginStatusList fail, need retry: %s", requestPayload)
		time.Sleep(time.Duration(2) * time.Second)
		_, err = util.HttpPost(url, requestPayload, "")
	}
	return err
}

// 检查并上报常驻插件状态
func (pm *PluginManager) CheckAndReportPlugin(pluginName, pluginVersion, cmdPath string, timeout int, env []string) (status string, err error) {
	exitCode := 0
	status = PERSIST_UNKNOWN
	exitCode, _, err = pm.executePlugin(cmdPath, []string{"--status"}, timeout, env, true)
	if err != nil {
		return
	}
	if exitCode != 0 {
		status = PERSIST_FAIL
	} else {
		status = PERSIST_RUNNING
	}
	return status, pm.ReportPluginStatus(pluginName, pluginVersion, status)
}

func needReportStatus(paramsList []string) bool {
	for _, p := range paramsList {
		for _, pp := range NEED_REFRESH_STATUS_API {
			if p == pp {
				return true
			}
		}
	}
	return false
}
