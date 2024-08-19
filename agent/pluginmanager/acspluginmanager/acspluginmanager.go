package acspluginmanager

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rodaine/table"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/metrics"
	. "github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"
	"github.com/aliyun/aliyun_assist_client/agent/util/versionutil"
	"github.com/aliyun/aliyun_assist_client/common/filelock"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/fuzzyjson"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/common/zipfile"
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
	AddSysTag         bool        `json:"addSysTag"`
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
			} else if pt == PLUGIN_COMMANDER {
				pc.pluginTypeStr = PLUGIN_COMMANDER
			} else {
				pc.pluginTypeStr = PLUGIN_UNKNOWN
			}
		case float64:
			pt, _ := pc.PluginType_.(float64)
			if pt == float64(PLUGIN_ONCE_INT) {
				pc.pluginTypeStr = PLUGIN_ONCE
			} else if pt == float64(PLUGIN_PERSIST_INT) {
				pc.pluginTypeStr = PLUGIN_PERSIST
			} else if pt == float64(PLUGIN_COMMANDER_INT){
				pc.pluginTypeStr = PLUGIN_COMMANDER
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
	PLUGINDIR, err = pathutil.GetPluginPath()
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
		OsType:     osutil.GetOsType(),
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
	var pluginInfoList []PluginInfo
	funcName := "List"
	exitCode = SUCCESS
	if local {
		if pluginName != "" {
			pluginInfoList, err = getInstalledPluginsByName(pluginName)
		} else {
			pluginInfoList, err = getAllInstalledPlugins()
		}
		if err != nil {
			exitCode, _ = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
			return
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
	pluginList, err := getAllInstalledPlugins()
	if err != nil {
		exitCode, _ = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
		return
	}
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

func (pm *PluginManager) ExecutePlugin(fetchOptions *ExecFetchOptions, executeParams *ExecuteParams) (exitCode int, err error) {
	log.GetLogger().Infoln("Enter ExecutePlugin")
	if pm.Verbose {
		log.GetLogger().WithFields(log.Fields{
			"fetchOptions":  fetchOptions,
			"executeParams": executeParams,
		}).Infof("ExecutePlugin")
	}

	if fetchOptions.File != "" {
		return pm.executePluginFromFile(fetchOptions.File, fetchOptions.FetchTimeoutInSeconds, executeParams)
	}
	// execute plugin exe-file
	return pm.executePluginOnlineOrLocal(fetchOptions, executeParams)
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

	idx, pluginInfo, err := getInstalledPluginNotRemovedByName(pluginName)
	if err != nil {
		exitCode, _ = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
		return
	}
	if pluginInfo == nil {
		exitCode, _ = errProcess(funcName, PACKAGE_NOT_FOUND, err, "plugin not exist "+pluginName)
		err = errors.New("Plugin " + pluginName + " not found in installed_plugins")
		return
	}

	var pluginLockFile *os.File
	pluginLockFile, err = openPluginLockFile(pluginName)
	if err != nil {
		err = NewOpenPluginLockFileError(err)
		exitCode, _ = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to remove plugin[%s]: %s", pluginName, err.Error()))
		return
	}
	defer pluginLockFile.Close()
	if err = filelock.TryLock(pluginLockFile); err != nil {
		err = NewAcquirePluginExclusiveLockError(err)
		exitCode, _ = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to remove plugin[%s]: %s", pluginName, err.Error()))
		return
	}
	defer filelock.Unlock(pluginLockFile)

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
	if err = updateInstalledPlugin(idx, pluginInfo); err != nil {
		exitCode, _ = errProcess(funcName, DUMP_INSTALLEDPLUGINS_ERR, err, "Update installed_plugins file err: "+err.Error())
		return
	}
	sysTagType := ""
	if pluginInfo.AddSysTag {
		sysTagType = RemoveSysTag
	}
	if err = pm.ReportPluginStatus(pluginInfo.Name, pluginInfo.Version, REMOVED, sysTagType); err != nil {
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
func (pm *PluginManager) executePluginFromFile(packagePath string, fetchTimeoutInSeconds int, executeParams *ExecuteParams) (exitCode int, err error) {
	const funcName = "ExecutePluginFromFile"
	log.GetLogger().Infof("Enter executePluginFromFile")
	var (
		// variables for metrics
		pluginName    string
		pluginVersion string
		resource      string = FetchFromLocalFile
		errorCode     string
		localArch     string
		pluginType    string

		// 执行插件时要注入的环境变量
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
		fmt.Println("Execute plugin from file: ", packagePath)
	}
	localArch, _ = GetArch()
	exitCode = SUCCESS
	if !fileutil.CheckFileIsExist(packagePath) {
		err = errors.New("File not exist: " + packagePath)
		exitCode, errorCode = errProcess(funcName, PACKAGE_NOT_FOUND, err, "Package file not exist: "+packagePath)
		return
	}

	const compressedConfigPath = "config.json"
	var configBody []byte
	configBody, err = zipfile.PeekFile(packagePath, compressedConfigPath)
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			errorMessage := "Package file isn`t a zip file: " + packagePath
			err = errors.New(errorMessage)
			exitCode, errorCode = errProcess(funcName, PACKAGE_FORMAT_ERR, err, errorMessage)
			return
		} else if errors.Is(err, os.ErrNotExist) {
			errorMessage := fmt.Sprintf("Manifest file config.json does not exist in %s.", packagePath)
			err = errors.New(errorMessage)
			exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, errorMessage)
			return
		} else {
			exitCode, errorCode = errProcess(funcName, UNZIP_ERR, err, fmt.Sprintf("Unzip err, file is [%s], target file is [%s], err is [%s]", packagePath, compressedConfigPath, err.Error()))
			return
		}
	}

	configContent := string(configBody)
	config := pluginConfig{}
	if err = fuzzyjson.Unmarshal(configContent, &config); err != nil {
		exitCode, errorCode = errProcess(funcName, UNMARSHAL_ERR, err, fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", configContent, err.Error()))
		return
	}

	pluginName = config.Name
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

	pluginIndex, plugin, err := getInstalledPluginByName(config.Name)
	if err != nil {
		exitCode, errorCode = errProcess(funcName, LOAD_INSTALLEDPLUGINS_ERR, err, "Load installed_plugins err: "+err.Error())
		return
	}
	if plugin != nil && plugin.IsRemoved {
		// 之前的同名插件已经被删除，相当于重新安装
		if err = deleteInstalledPluginByIndex(pluginIndex); err != nil {
			exitCode, errorCode = errProcess(funcName, DUMP_INSTALLEDPLUGINS_ERR, err, "Update installed_plugins file err: "+err.Error())
			return
		}

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

	// Acquire plugin-wise shared lock
	var pluginLockFile *os.File
	pluginLockFile, err = openPluginLockFile(config.Name)
	if err != nil {
		err = NewOpenPluginLockFileError(err)
		exitCode, errorCode = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to execute plugin[%s %s]: %s", config.Name, config.Version, err.Error()))
		return
	}
	defer pluginLockFile.Close()
	if err = filelock.TryRLock(pluginLockFile); err != nil {
		err = NewAcquirePluginExclusiveLockError(err)
		exitCode, errorCode = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to execute plugin[%s %s]: %s", config.Name, config.Version, err.Error()))
		return
	}
	defer filelock.Unlock(pluginLockFile)

	var fetched *Fetched
	var exitingErr ExitingError
	fetchTimeout := time.Duration(fetchTimeoutInSeconds) * time.Second
	guardErr := InstallLockGuard{
		PluginName:    config.Name,
		PluginVersion: config.Version,

		AcquireSharedLockTimeout: fetchTimeout,
	}.Do(func() {
		// Double-check if specified plugin has been installed by another
		// plugin-manager process, that releases plugin-version-wise
		// exclusive lock and then this acquired.
		fetched, exitingErr = queryFromLocalOnly(config.Name, config.Version)
		if exitingErr == nil || !errors.Is(exitingErr, ErrPackageNotFound) {
			return
		}

		fetched, exitingErr = installFromFile(packagePath, &config, plugin, pluginIndex, fetchTimeout)
	}, func() {
		fetched, exitingErr = queryFromLocalOnly(config.Name, config.Version)
		// TODO-FIXME: envPrePluginDir SHOULD be constructed on demand only
		fetched.EnvPrePluginDir = envPrePluginDir
	})
	if guardErr != nil {
		err = guardErr
		exitCode, errorCode = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to execute plugin[%s %s]: %s", config.Name, config.Version, err.Error()))
		return
	}
	if exitingErr != nil {
		err = exitingErr.Unwrap()
		exitCode, errorCode = errProcess(funcName, exitingErr.ExitCode(), exitingErr.Unwrap(), exitingErr.Error())
		return
	}
	fmt.Printf("Plugin[%s] installed!\n", fetched.PluginName)

	pluginName = fetched.PluginName
	pluginVersion = fetched.PluginVersion
	pluginType = fetched.PluginType

	executionTimeoutInSeconds := fetched.ExecutionTimeoutInSeconds
	if executeParams.OptionalExecutionTimeoutInSeconds != nil {
		executionTimeoutInSeconds = *executeParams.OptionalExecutionTimeoutInSeconds
	}

	args := executeParams.SplitArgs()
	env := []string{
		"PLUGIN_DIR=" + fetched.EnvPluginDir,
		"PRE_PLUGIN_DIR=" + fetched.EnvPrePluginDir,
	}
	var options []process.CmdOption
	options, err = prepareCmdOptions(executeParams)
	if err != nil {
		exitCode, errorCode = errProcess(funcName, EXECUTE_FAILED, err, fmt.Sprintf("Failed to set execution options for plugin[%s %s]: %s", pluginName, pluginVersion, err.Error()))
		return
	}
	exitCode, errorCode, err = pm.executePlugin(fetched.Entrypoint, args, executionTimeoutInSeconds, env, false, options...)
	// 常驻插件和一次性插件：需要主动上报一次插件状态
	if pluginType == PLUGIN_PERSIST {
		status, err := pm.CheckAndReportPlugin(pluginName, pluginVersion, fetched.Entrypoint, executionTimeoutInSeconds, env, "")
		log.GetLogger().WithFields(log.Fields{
			"pluginName":    pluginName,
			"pluginVersion": pluginVersion,
			"cmdPath":       fetched.Entrypoint,
			"timeout":       executionTimeoutInSeconds,
			"env":           env,
			"status":        status,
		}).WithError(err).Infof("CheckAndReportPlugin")
	} else if pluginType == PLUGIN_ONCE {
		pm.ReportPluginStatus(pluginName, pluginVersion, ONCE_INSTALLED, "")
	}
	return
}

func installFromFile(packagePath string, config *pluginConfig, plugin *PluginInfo, pluginIndex int, timeout time.Duration) (*Fetched, ExitingError) {
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var envPrePluginDir string

	if pluginIndex == -1 {
		plugin = &PluginInfo{
			Timeout: "60",
		}
	} else {
		envPrePluginDir = filepath.Join(PLUGINDIR, plugin.Name, plugin.Version)
	}

	executionTimeoutInSeconds := 60
	if t, err := strconv.Atoi(config.Timeout); err != nil {
		config.Timeout = plugin.Timeout
	} else {
		executionTimeoutInSeconds = t
	}

	plugin.Name = config.Name
	plugin.Arch = config.Arch
	plugin.OSType = config.OsType
	plugin.RunPath = config.RunPath
	plugin.Timeout = config.Timeout
	plugin.Publisher = config.Publisher
	plugin.Version = config.Version
	plugin.AddSysTag = config.AddSysTag
	plugin.SetPluginType(config.PluginType())
	plugin.Url = "local"
	if config.HeartbeatInterval <= 0 {
		plugin.HeartbeatInterval = 60
	} else {
		plugin.HeartbeatInterval = config.HeartbeatInterval
	}

	// TODO-FIXME: Calculating the MD5 checksum for plugin package is also a
	// time-consuming procedure, which should be cancelable when timed out
	md5Checksum, err := util.ComputeMd5(packagePath)
	if err != nil {
		return nil, NewMD5CheckExitingError(err, "Compute md5 of plugin file err: "+err.Error())
	}
	plugin.Md5 = md5Checksum

	pluginPath := filepath.Join(PLUGINDIR, plugin.Name, plugin.Version)
	pathutil.MakeSurePath(pluginPath)
	if err := zipfile.UnzipContext(ctx, packagePath, pluginPath, false); err != nil {
		return nil, NewUnzipExitingError(err, fmt.Sprintf("Unzip err, file is [%s], target dir is [%s], err is [%s]", packagePath, pluginPath, err.Error()))
	}

	if config.PluginType() == PLUGIN_COMMANDER {
		commanderInfoPath := filepath.Join(pluginPath, "axt-commander.json")
		if !fileutil.CheckFileIsExist(commanderInfoPath) {
			err := errors.New(fmt.Sprintf("File axt-commander.json not exist, %s.", commanderInfoPath))
			return nil, NewPluginFormatExitingError(err, fmt.Sprintf("File axt-commander.json not exist, %s.", commanderInfoPath))
		}
		commanderInfo := CommanderInfo{}
		if content, err := fuzzyjson.UnmarshalFile(commanderInfoPath, &commanderInfo); err != nil {
			return nil, NewUnmarshalExitingError(err, fmt.Sprintf("Unmarshal axt-commander.json err, axt-commander.json is [%s], err is [%s]", string(content), err.Error()))
		}
		plugin.CommanderInfo = commanderInfo
	}

	cmdPath := filepath.Join(pluginPath, config.RunPath)
	if !fileutil.CheckFileIsExist(cmdPath) {
		log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
		return nil, NewPluginFormatExitingError(errors.New("Cmd file not exist: "+cmdPath), fmt.Sprintf("Executable file not exist, %s.", cmdPath))
	}
	if osutil.GetOsType() != osutil.OSWin {
		if err := os.Chmod(cmdPath, os.FileMode(0o744)); err != nil {
			return nil, NewExecutablePermissionExitingError(err, "Make plugin file executable err: "+err.Error())
		}
	}
	if pluginIndex == -1 {
		plugin.PluginID = "local_" + plugin.Name + "_" + plugin.Version
		_, err = insertNewInstalledPlugin(plugin)
	} else {
		err = updateInstalledPlugin(pluginIndex, plugin)
	}
	if err != nil {
		return nil, NewDumpInstalledPluginsExitingError(err)
	}

	return &Fetched{
		PluginName:    config.Name,
		PluginVersion: config.Version,
		PluginType:    config.PluginType(),
		AddSysTag:     config.AddSysTag,

		Entrypoint:                cmdPath,
		ExecutionTimeoutInSeconds: executionTimeoutInSeconds,
		EnvPluginDir:              pluginPath,
		EnvPrePluginDir:           envPrePluginDir,
	}, nil
}

func (pm *PluginManager) executePluginOnlineOrLocal(fetchOptions *ExecFetchOptions, executeParams *ExecuteParams) (exitCode int, err error) {
	const funcName = "ExecutePluginOnlineOrLocal"
	log.GetLogger().Info("Enter executePluginOnlineOrLocal")
	var (
		// variables for metrics
		resource  string
		errorCode string
		localArch string

		pluginName    string = fetchOptions.PluginName
		pluginVersion string = fetchOptions.Version
		pluginType    string = PLUGIN_ONCE
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
	localArch, _ = GetArch()

	var fetched *Fetched
	var queried *PluginInfo
	var exitingErr ExitingError
	if fetchOptions.Local {
		// execute local plugin
		fetched, exitingErr = queryFromLocalOnly(fetchOptions.PluginName, fetchOptions.Version)
	} else {
		fetched, queried, exitingErr = queryFromOnlineOrLocal(fetchOptions, localArch)
	}
	if exitingErr != nil {
		err = exitingErr.Unwrap()
		exitCode, errorCode = errProcess(funcName, exitingErr.ExitCode(), exitingErr.Unwrap(), exitingErr.Error())
		return
	}

	// Acquire plugin-wise shared lock
	var pluginLockFile *os.File
	pluginLockFile, err = openPluginLockFile(fetchOptions.PluginName)
	if err != nil {
		err = NewOpenPluginLockFileError(err)
		exitCode, errorCode = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to execute plugin[%s %s]: %s", fetchOptions.PluginName, fetchOptions.Version, err.Error()))
		return
	}
	defer pluginLockFile.Close()
	if err = filelock.TryRLock(pluginLockFile); err != nil {
		err = NewAcquirePluginExclusiveLockError(err)
		exitCode, errorCode = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to execute plugin[%s %s]: %s", fetchOptions.PluginName, fetchOptions.Version, err.Error()))
		return
	}
	defer filelock.Unlock(pluginLockFile)

	if fetched != nil {
		resource = FetchFromLocalInstalled
	} else if queried != nil {
		resource = FetchFromOnline
		fetchTimeout := time.Duration(fetchOptions.FetchTimeoutInSeconds) * time.Second
		guardErr := InstallLockGuard{
			PluginName:    queried.Name,
			PluginVersion: queried.Version,

			AcquireSharedLockTimeout: fetchTimeout,
		}.Do(func() {
			// Double-check if specified plugin has been installed by another
			// plugin-manager process, that releases plugin-version-wise
			// exclusive lock and then this acquired.
			fetched, exitingErr = queryFromLocalOnly(queried.Name, queried.Version)
			if exitingErr == nil || !errors.Is(exitingErr, ErrPackageNotFound) {
				return
			}

			fetched, exitingErr = installFromOnline(queried, fetchTimeout, localArch)
		}, func() {
			fetched, exitingErr = queryFromLocalOnly(queried.Name, queried.Version)
		})
		if guardErr != nil {
			err = guardErr
			exitCode, errorCode = errProcess(funcName, LOCKING_ERR, err, fmt.Sprintf("Failed to execute plugin[%s %s]: %s", queried.Name, queried.Version, err.Error()))
			return
		}
		if exitingErr != nil {
			err = exitingErr.Unwrap()
			exitCode, errorCode = errProcess(funcName, exitingErr.ExitCode(), exitingErr.Unwrap(), exitingErr.Error())
			return
		}
	} else {
		// SHOULD NEVER RUN INTO THIS BRANCH
		err = ErrPackageNotFound
		exitCode, errorCode = errProcess(funcName, PACKAGE_NOT_FOUND, err, err.Error())
		return
	}

	pluginName = fetched.PluginName
	pluginVersion = fetched.PluginVersion
	pluginType = fetched.PluginType

	executionTimeoutInSeconds := fetched.ExecutionTimeoutInSeconds
	if executeParams.OptionalExecutionTimeoutInSeconds != nil {
		executionTimeoutInSeconds = *executeParams.OptionalExecutionTimeoutInSeconds
	}
	args := executeParams.SplitArgs()
	env := []string{
		"PLUGIN_DIR=" + fetched.EnvPluginDir,
		"PRE_PLUGIN_DIR=" + fetched.EnvPrePluginDir,
	}
	var options []process.CmdOption
	options, err = prepareCmdOptions(executeParams)
	if err != nil {
		exitCode, errorCode = errProcess(funcName, EXECUTE_FAILED, err, fmt.Sprintf("Failed to set execution options for plugin[%s %s]: %s", pluginName, pluginVersion, err.Error()))
		return
	}
	exitCode, errorCode, err = pm.executePlugin(fetched.Entrypoint, args, executionTimeoutInSeconds, env, false, options...)
	// 常驻插件：如果是从线上拉取的插件，或者调用的接口有可能改变插件状态，需要主动上报一次插件状态
	// 一次性插件：如果插件是从线上拉取的，需要上报一次插件状态
	sysTagType := ""
	if pluginType == PLUGIN_PERSIST && (needReportStatus(args) || resource == FetchFromOnline) {
		if resource == FetchFromOnline {
			if fetched.AddSysTag {
				sysTagType = AddSysTag
			} else if len(fetched.EnvPrePluginDir) != 0 {
				sysTagType = RemoveSysTag
			}
		}
		status, err := pm.CheckAndReportPlugin(pluginName, pluginVersion, fetched.Entrypoint, executionTimeoutInSeconds, env, sysTagType)
		log.GetLogger().WithFields(log.Fields{
			"pluginName":    pluginName,
			"pluginVersion": pluginVersion,
			"cmdPath":       fetched.Entrypoint,
			"timeout":       executionTimeoutInSeconds,
			"env":           env,
			"status":        status,
		}).WithError(err).Infof("CheckAndReportPlugin")
	} else if pluginType == PLUGIN_ONCE && resource == FetchFromOnline {
		if fetched.AddSysTag {
			sysTagType = AddSysTag
		} else if len(fetched.EnvPrePluginDir) != 0 {
			sysTagType = RemoveSysTag
		}
		pm.ReportPluginStatus(pluginName, pluginVersion, ONCE_INSTALLED, sysTagType)
	}
	return
}

func queryFromLocalOnly(pluginName string, pluginVersion string) (*Fetched, ExitingError) {
	localInfo, err := getLocalPluginInfo(pluginName, pluginVersion)
	if err != nil {
		return nil, NewLoadInstalledPluginsExitingError(err)
	}
	if localInfo == nil {
		return nil, NewPackageNotFoundExitingError(ErrPackageNotFound, fmt.Sprintf("Could not found local package [%s]", pluginName))
	}

	envPluginDir := filepath.Join(PLUGINDIR, localInfo.Name, localInfo.Version)
	executionTimeoutInSeconds := 60
	if t, err := strconv.Atoi(localInfo.Timeout); err == nil {
		executionTimeoutInSeconds = t
	}
	return &Fetched{
		PluginName:    localInfo.Name,
		PluginVersion: localInfo.Version,
		PluginType:    localInfo.PluginType(),
		AddSysTag:     localInfo.AddSysTag,

		Entrypoint:                filepath.Join(envPluginDir, localInfo.RunPath),
		ExecutionTimeoutInSeconds: executionTimeoutInSeconds,
		EnvPluginDir:              envPluginDir,
		EnvPrePluginDir:           "",
	}, nil
}

// didn't set --local, so local & online both try
func queryFromOnlineOrLocal(fetchOptions *ExecFetchOptions, localArch string) (*Fetched, *PluginInfo, ExitingError) {
	localInfo, err := getLocalPluginInfo(fetchOptions.PluginName, fetchOptions.Version)
	if err != nil {
		return nil, nil, NewLoadInstalledPluginsExitingError(err)
	}

	onlineInfo, onlineOtherArch, err := getOnlinePluginInfo(fetchOptions.PluginName, fetchOptions.Version)
	if err != nil {
		return nil, nil, NewGetOnlinePackageInfoExitingError(err, "Get plugin info from online err: "+err.Error())
	}

	useLocal := false
	if localInfo != nil {
		if onlineInfo != nil {
			// 本地和线上版本一致，使用本地插件文件
			if versionutil.CompareVersion(localInfo.Version, onlineInfo.Version) == 0 {
				log.GetLogger().Infof("ExecutePluginOnlineOrLocal: Plugin[%s], local version[%s] same to online version[%s], so use local package", fetchOptions.PluginName, localInfo.Version, onlineInfo.Version)
				useLocal = true
			} else {
				// 本地和线上版本不一致，使用线上版本
				log.GetLogger().Infof("ExecutePluginOnlineOrLocal: Plugin[%s], local version[%s] different from online version[%s], so use online package", fetchOptions.PluginName, localInfo.Version, onlineInfo.Version)
			}
		} else {
			useLocal = true
		}
	} else {
		if onlineInfo == nil {
			var tip string
			if len(onlineOtherArch) == 0 {
				tip = fmt.Sprintf("Could not found both local and online, package[%s] version[%s]\n", fetchOptions.PluginName, fetchOptions.Version)
			} else {
				tip = fmt.Sprintf("Could not found local package[%s] version[%s], found online package but it`s arch[%s] not match local_arch[%s] \n", fetchOptions.PluginName, fetchOptions.Version, strings.Join(onlineOtherArch, ", "), localArch)
			}
			return nil, nil, NewPackageNotFoundExitingError(ErrPackageNotFound, tip)
		}
	}

	// use local package
	if useLocal {
		executionTimeoutInSeconds := 60
		if t, err := strconv.Atoi(localInfo.Timeout); err == nil {
			executionTimeoutInSeconds = t
		}

		pluginPath := filepath.Join(PLUGINDIR, localInfo.Name, localInfo.Version)
		return &Fetched{
			PluginName:    localInfo.Name,
			PluginVersion: localInfo.Version,
			PluginType:    localInfo.PluginType(),

			Entrypoint:                filepath.Join(pluginPath, localInfo.RunPath),
			ExecutionTimeoutInSeconds: executionTimeoutInSeconds,
			EnvPluginDir:              pluginPath,
			EnvPrePluginDir:           "",
		}, nil, nil
	}

	return nil, onlineInfo, nil
}

// 下载并安装插件
// pull package
func installFromOnline(onlineInfo *PluginInfo, timeout time.Duration, localArch string) (*Fetched, ExitingError) {
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	filePath := filepath.Join(PLUGINDIR, onlineInfo.Name+".zip")

	log.GetLogger().Infof("Downloading package from [%s], save to [%s] ", onlineInfo.Url, filePath)
	const maxRetries = 3
	const retryDelay = 3 * time.Second
	var err error
	if timeout == 0 {
		err = func(remoteUrl string, localPath string) error {
			var err2 error
			for retry := maxRetries; retry > 0; retry-- {
				err2 = util.HttpDownlod(remoteUrl, localPath)
				if err2 == nil {
					break
				}
				time.Sleep(retryDelay)
			}

			return err2
		}(onlineInfo.Url, filePath)
	} else {
		err = func(ctx context.Context, remoteUrl string, localPath string) error {
			var err2 error
			for retry := maxRetries; retry > 0; retry-- {
				err2 = util.HttpDownloadContext(ctx, remoteUrl, localPath)
				if err2 == nil {
					break
				}

				delayTimer := time.NewTimer(retryDelay)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-delayTimer.C:
					continue
				}
			}
			return err2
		}(ctx, onlineInfo.Url, filePath)
	}
	if err != nil {
		return nil, NewDownloadExitingError(err, fmt.Sprintf("Downloading package failed, plugin.Url is [%s], err is [%s]", onlineInfo.Url, err.Error()))
	}

	log.GetLogger().Infoln("Check MD5...")
	// TODO-FIXME: Calculating the MD5 checksum for plugin package is also a
	// time-consuming procedure, which should be cancelable when timed out
	md5Checksum, err := util.ComputeMd5(filePath)
	if err != nil {
		return nil, NewMD5CheckExitingError(err, fmt.Sprintf("Compute md5 of plugin file[%s] err, plugin.Url is [%s], err is [%s]", filePath, onlineInfo.Url, err.Error()))
	}
	if strings.ToLower(md5Checksum) != strings.ToLower(onlineInfo.Md5) {
		log.GetLogger().Errorf("Md5 not match, onlineInfo.Md5[%s], package file md5[%s]\n", onlineInfo.Md5, md5Checksum)
		return nil, NewMD5CheckExitingError(errors.New("Md5 not macth"), fmt.Sprintf("Md5 not match, onlineInfo.Md5 is [%s], real md5 is [%s], plugin.Url is [%s]", onlineInfo.Md5, md5Checksum, onlineInfo.Url))
	}

	unzipdir := filepath.Join(PLUGINDIR, onlineInfo.Name, onlineInfo.Version)
	pathutil.MakeSurePath(unzipdir)
	log.GetLogger().Infoln("Unzip package...")
	if err := zipfile.UnzipContext(ctx, filePath, unzipdir, false); err != nil {
		return nil, NewUnzipExitingError(err, fmt.Sprintf("Unzip package err, plugin.Url is [%s], err is [%s]", onlineInfo.Url, err.Error()))
	}
	os.RemoveAll(filePath)

	config_path := filepath.Join(unzipdir, "config.json")
	if !fileutil.CheckFileIsExist(config_path) {
		err := errors.New(fmt.Sprintf("File config.json not exist, %s.", config_path))
		return nil, NewPluginFormatExitingError(err, fmt.Sprintf("File config.json not exist, %s.", config_path))
	}

	config := pluginConfig{}
	if content, err := fuzzyjson.UnmarshalFile(config_path, &config); err != nil {
		return nil, NewUnmarshalExitingError(err, fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", string(content), err.Error()))
	}
	if config.HeartbeatInterval <= 0 {
		config.HeartbeatInterval = 60
	}
	if config.PluginType() != onlineInfo.PluginType() {
		tip := fmt.Sprintf("config.PluginType[%s] not match to pluginType[%s]", config.PluginType(), onlineInfo.PluginType())
		return nil, NewPluginFormatExitingError(errors.New(tip), tip)
	}
	// 接口返回的插件信息中没有HeartbeatInterval字段，需要以插件包中的config.json为准
	onlineInfo.HeartbeatInterval = config.HeartbeatInterval
	onlineInfo.SetPluginType(config.PluginType())
	onlineInfo.AddSysTag = config.AddSysTag

	if config.PluginType() == PLUGIN_COMMANDER {
		commanderInfoPath := filepath.Join(unzipdir, "axt-commander.json")
		if !fileutil.CheckFileIsExist(commanderInfoPath) {
			err := errors.New(fmt.Sprintf("File axt-commander.json not exist, %s.", commanderInfoPath))
			return nil, NewPluginFormatExitingError(err, fmt.Sprintf("File axt-commander.json not exist, %s.", commanderInfoPath))
		}
		commanderInfo := CommanderInfo{}
		if content, err := fuzzyjson.UnmarshalFile(commanderInfoPath, &commanderInfo); err != nil {
			return nil, NewUnmarshalExitingError(err, fmt.Sprintf("Unmarshal axt-commander.json err, axt-commander.json is [%s], err is [%s]", string(content), err.Error()))
		}
		onlineInfo.CommanderInfo = commanderInfo
	}

	cmdPath := filepath.Join(unzipdir, config.RunPath)
	// 检查系统类型和架构是否符合
	if strings.ToLower(onlineInfo.OSType) != "both" && strings.ToLower(onlineInfo.OSType) != osutil.GetOsType() {
		err := errors.New("Plugin ostype not suit for this system")
		return nil, NewPluginFormatExitingError(err, fmt.Sprintf("Plugin ostype[%s] not suit for this system[%s], plugin.Url is [%s]", onlineInfo.OSType, osutil.GetOsType(), onlineInfo.Url))
	}
	if strings.ToLower(onlineInfo.Arch) != "all" && strings.ToLower(onlineInfo.Arch) != localArch {
		err := errors.New("Plugin arch not suit for this system")
		return nil, NewPluginFormatExitingError(err, fmt.Sprintf("Plugin arch[%s] not suit for this system[%s], plugin.Url is [%s]", onlineInfo.Arch, localArch, onlineInfo.Url))
	}
	if !fileutil.CheckFileIsExist(cmdPath) {
		log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
		err := errors.New("Cmd file not exist: " + cmdPath)
		return nil, NewPluginFormatExitingError(err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
	}
	if osutil.GetOsType() != osutil.OSWin {
		if err = os.Chmod(cmdPath, os.FileMode(0o744)); err != nil {
			return nil, NewExecutablePermissionExitingError(err, fmt.Sprintf("Make plugin file executable err, plugin.Url is [%s], err is [%s]", onlineInfo.Url, err.Error()))
		}
	}

	// update INSTALLEDPLUGINS file
	var envPrePluginDir string
	pluginIndex, pluginInfo, err := getInstalledPluginByName(onlineInfo.Name)
	if err != nil {
		return nil, NewLoadInstalledPluginsExitingError(err)
	}
	if pluginIndex != -1 && pluginInfo.IsRemoved {
		// Actually from the database remove the record of removed plugin
		if err = deleteInstalledPluginByIndex(pluginIndex); err != nil {
			return nil, NewDumpInstalledPluginsExitingError(err)
		}

		pluginIndex = -1
	}
	if pluginIndex == -1 {
		_, err = insertNewInstalledPlugin(onlineInfo)
	} else {
		envPrePluginDir = filepath.Join(PLUGINDIR, pluginInfo.Name, pluginInfo.Version)
		err = updateInstalledPlugin(pluginIndex, onlineInfo)
	}
	if err != nil {
		return nil, NewDumpInstalledPluginsExitingError(err)
	}

	executionTimeoutInSeconds := 60
	if t, err := strconv.Atoi(config.Timeout); err == nil {
		executionTimeoutInSeconds = t
	}
	return &Fetched{
		PluginName:    config.Name,
		PluginVersion: config.Version,
		PluginType:    config.PluginType(),
		AddSysTag:     config.AddSysTag,

		Entrypoint:                cmdPath,
		ExecutionTimeoutInSeconds: executionTimeoutInSeconds,
		EnvPluginDir:              filepath.Join(PLUGINDIR, config.Name, config.Version),
		EnvPrePluginDir:           envPrePluginDir,
	}, nil
}

func (pm *PluginManager) executePlugin(cmdPath string, paramList []string, timeout int, env []string, quiet bool, options ...process.CmdOption) (exitCode int, errorCode string, err error) {
	log.GetLogger().Infof("Enter executePlugin, cmdPath[%s] paramList[%v] paramCount[%d] timeout[%d]\n", cmdPath, paramList, len(paramList), timeout)
	funcName := "ExecutePlugin"
	if !fileutil.CheckFileIsExist(cmdPath) {
		log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
		err = errors.New("Cmd file not exist: " + cmdPath)
		exitCode, errorCode = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
		return
	}
	if pm.Verbose {
		fmt.Printf("Run cmd: %s, params: %v\n", cmdPath, paramList)
	}

	processCmd := process.NewProcessCmd(options...)
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

func (pm *PluginManager) VerifyPlugin(fetchOptions *VerifyFetchOptions, executeParams *ExecuteParams) (exitCode int, err error) {
	const funcName = "VerifyPlugin"
	log.GetLogger().WithFields(log.Fields{
		"fetchOptions":  fetchOptions,
		"executeParams": executeParams,
	}).Infoln("Enter VerifyPlugin")

	// pull package
	unzipdir := filepath.Join(PLUGINDIR, "verify_plugin_test")
	exitCode, err = func(packageUrl string, timeoutInSeconds int) (int, error) {
		ctx := context.Background()
		var cancel context.CancelFunc
		if timeoutInSeconds > 0 {
			ctx, cancel = context.WithTimeout(ctx, time.Duration(timeoutInSeconds)*time.Second)
			defer cancel()
		}

		log.GetLogger().Infoln("Downloading package from ", packageUrl)
		filePath, err := func(packageUrl string, timeoutInSeconds int) (string, error) {
			fileName := packageUrl[strings.LastIndex(packageUrl, "/")+1:]
			filePath := PLUGINDIR + Separator + fileName
			if len(packageUrl) > 4 && packageUrl[:4] == "http" {
				return filePath, util.HttpDownloadContext(ctx, packageUrl, filePath)
			} else {
				return filePath, FileProtocolDownload(packageUrl, filePath)
			}
		}(packageUrl, timeoutInSeconds)
		if err != nil {
			exitcode, _ := errProcess(funcName, DOWNLOAD_FAIL, err, fmt.Sprintf("Downloading package failed, url is [%s], err is [%s]", packageUrl, err.Error()))
			return exitcode, err
		}

		pathutil.MakeSurePath(unzipdir)
		log.GetLogger().Infoln("Unzip package...")
		if err := zipfile.UnzipContext(ctx, filePath, unzipdir, false); err != nil {
			exitcode, _ := errProcess(funcName, UNZIP_ERR, err, fmt.Sprintf("Unzip package err, url is [%s], err is [%s]", packageUrl, err.Error()))
			return exitcode, err
		}

		os.RemoveAll(filePath)

		return 0, nil
	}(fetchOptions.Url, fetchOptions.FetchTimeoutInSeconds)
	if err != nil {
		return
	}

	configPath := filepath.Join(unzipdir, "config.json")
	if !fileutil.CheckFileIsExist(configPath) {
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
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin ostype[%s] not suit for this system[%s], url is [%s]", config.OsType, osutil.GetOsType(), fetchOptions.Url))
		return
	}
	localArch, _ := GetArch()
	if config.Arch != "" && strings.ToLower(config.Arch) != "all" && strings.ToLower(config.Arch) != localArch {
		err = errors.New("Plugin arch not suit for this system")
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Plugin arch[%s] not suit for this system[%s], url is [%s]", config.Arch, localArch, fetchOptions.Url))
		return
	}

	cmdPath := filepath.Join(unzipdir, config.RunPath)
	if !fileutil.CheckFileIsExist(cmdPath) {
		err = errors.New("Can not find the cmd file")
		exitCode, _ = errProcess(funcName, PLUGIN_FORMAT_ERR, err, fmt.Sprintf("Executable file not exist, %s.", cmdPath))
		return
	}
	if osutil.GetOsType() != osutil.OSWin {
		if err = os.Chmod(cmdPath, os.FileMode(0o744)); err != nil {
			exitCode, _ = errProcess(funcName, EXECUTABLE_PERMISSION_ERR, err, "Make plugin file executable err: "+err.Error())
			return
		}
	}

	executionTimeoutInSeconds := 60
	if t, err := strconv.Atoi(config.Timeout); err != nil {
		fmt.Println("config.Timeout is invalid: ", config.Timeout)
	} else {
		executionTimeoutInSeconds = t
	}
	if executeParams.OptionalExecutionTimeoutInSeconds != nil {
		executionTimeoutInSeconds = *executeParams.OptionalExecutionTimeoutInSeconds
	}

	// 执行插件时要注入的环境变量
	env := []string{
		// 当前执行的插件的执行目录
		"PLUGIN_DIR=" + unzipdir,
		// 如果已有同名插件，表示已有同名插件的执行目录；否则为空
		"PRE_PLUGIN_DIR=",
	}
	var options []process.CmdOption
	options, err = prepareCmdOptions(executeParams)
	if err != nil {
		exitCode, _ = errProcess(funcName, EXECUTE_FAILED, err, fmt.Sprintf("Failed to set execution options: %s", err.Error()))
		return
	}
	exitCode, _, err = pm.executePlugin(cmdPath, executeParams.SplitArgs(), executionTimeoutInSeconds, env, false, options...)
	return
}

// 向服务端上报某个插件状态
func (pm *PluginManager) ReportPluginStatus(pluginName, pluginVersion, status string, sysTagType string) error {
	if len(pluginName) > PLUGIN_NAME_MAXLEN {
		pluginName = pluginName[:PLUGIN_NAME_MAXLEN]
	}
	if len(pluginVersion) > PLUGIN_VERSION_MAXLEN {
		pluginVersion = pluginVersion[:PLUGIN_VERSION_MAXLEN]
	}
	pluginStatusRequest := PluginStatusResquest{
		Plugin: []PluginStatus{
			{
				Name:       pluginName,
				Version:    pluginVersion,
				Status:     status,
				SysTagType: sysTagType,
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
func (pm *PluginManager) CheckAndReportPlugin(pluginName, pluginVersion, cmdPath string, timeout int, env []string, sysTagType string) (status string, err error) {
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
	return status, pm.ReportPluginStatus(pluginName, pluginVersion, status, sysTagType)
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

func InstallPluginFromOnline(onlineInfo *PluginInfo, timeout int) error {
	localArch, _ := GetArch()
	_, err := installFromOnline(onlineInfo, time.Second*time.Duration(timeout), localArch)
	return err
}

func QueryPluginFromOnline(pluginName, pluginType, version string) (*PluginInfo, error) {
	pluginInfos, err := getPackageInfo(pluginName, version, true)
	if err != nil {
		return nil, err
	}
	if len(pluginInfos) == 0 {
		return nil, fmt.Errorf("not found")
	}
	var res *PluginInfo
	for i, _ := range pluginInfos {
		if pluginInfos[i].PluginType() != pluginType {
			continue
		}
		if version != "" {
			if pluginInfos[i].Version == version {
				res = &pluginInfos[i]
				break
			}
		} else {
			if res == nil || versionutil.CompareVersion(pluginInfos[i].Version, res.Version) > 0 {
				res = &pluginInfos[i]
			}
		}
	}
	if res == nil {
		return nil, fmt.Errorf("not found")
	}
	return res, nil
}

func QueryPluginFromLocal(pluginName, pluginType string) (*PluginInfo, error) {
	plugins, err := getInstalledPluginsByName(pluginName)
	if err != nil {
		return nil, err
	}
	var res *PluginInfo
	for i, _ := range plugins {
		if plugins[i].IsRemoved || plugins[i].PluginType() != pluginType {
			continue
		}
	}
	if res == nil {
		return nil, fmt.Errorf("not found")
	}
	return res, nil
}

func LoadAllPluginFromLocal(pluginType string) ([]PluginInfo, error) {
	plugins, err := getAllInstalledPlugins()
	if err != nil {
		return nil, err
	}
	if pluginType != "" {
		res := []PluginInfo{}
		for _, p := range plugins {
			if p.PluginType() == pluginType && !p.IsRemoved {
				res = append(res, p)
			}
		}
		return res, nil
	}
	return plugins, nil
}
