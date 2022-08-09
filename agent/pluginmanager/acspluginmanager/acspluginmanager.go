package acspluginmanager

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util/process"

	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/thirdparty/table"
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager/acspluginmanager/thirdparty/shlex"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/agent/util/osutil"
	"github.com/aliyun/aliyun_assist_client/agent/util/versionutil"
)

const (
	// plugin type
	PLUGIN_ONCE    = 0
	PLUGIN_PERSIST = 1

	//plugin status
	PERSIST_RUNNING = "PERSIST_RUNNING"
	PERSIST_FAIL    = "PERSIST_FAIL"
	PERSIST_UNKNOWN = "PERSIST_UNKNOWN"
	ONCE_INSTALLED  = "ONCE_INSTALLED"
)

type pluginInfo struct {
	PluginId       string `json:"pluginId"`
	Name           string `json:"name"`
	Arch           string `json:"arch"`
	OsType         string `json:"osType"`
	Version        string `json:"version"`
	Publisher      string `json:"publisher"`
	Url            string `json:"url"`
	Md5            string `json:"md5"`
	RunPath        string `json:"runPath"`
	Timeout        string `json:"timeout"`
	IsPreInstalled string `json:"isPreInstalled"`
	PluginType     int    `json:"pluginType"`
}

type pluginConfig struct {
	Name       string `json:"name"`
	Arch       string `json:"arch"`
	OsType     string `json:"osType"`
	RunPath    string `json:"runPath"`
	Timeout    string `json:"timeout"`
	Publisher  string `json:"publisher"`
	Version    string `json:"version"`
	PluginType int    `json:"pluginType"`
}

type pluginStatus struct {
	PluginId string `json:"pluginId"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Version  string `json:"version"`
	OsType   string `json:"os"`
	Arch     string `json:"arch"`
}

type InstalledPlugins struct {
	PluginList []pluginInfo `json:"pluginList"`
}

type ListRet struct {
	Code       int          `json:"code"`
	RequestId  string       `json:"requestId"`
	InstanceId string       `json:"instanceId"`
	PluginList []pluginInfo `json:"pluginList"`
}

type PluginManager struct {
	Verbose bool
	Yes     bool
}

var INSTALLEDPLUGINS string
var PLUGINDIR string

const Separator = string(filepath.Separator)

func NewPluginManager(verbose bool) (*PluginManager, error) {
	var err error
	PLUGINDIR, err = util.GetPluginPath()
	INSTALLEDPLUGINS = PLUGINDIR + Separator + "installed_plugins"
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &PluginManager{
		Verbose: verbose,
		Yes: true,
	}, nil
}

func loadInstalledPlugins() ([]pluginInfo, error) {
	installedPlugins := InstalledPlugins{}
	if util.CheckFileIsExist(INSTALLEDPLUGINS) {
		if _, err := unmarshalFile(INSTALLEDPLUGINS, &installedPlugins); err != nil {
			return nil, err
		}
		return installedPlugins.PluginList, nil
	}
	return installedPlugins.PluginList, nil
}

func dumpInstalledPlugins(pluginInfoList []pluginInfo) error {
	installedPlugins := InstalledPlugins{
		PluginList: pluginInfoList,
	}
	pluginInfoListStr, err := marshal(&installedPlugins)
	if err != nil {
		return err
	}
	err = util.WriteStringToFile(INSTALLEDPLUGINS, pluginInfoListStr)
	return err
}

func printPluginInfo(pluginInfoList *[]pluginInfo) {
	tbl := table.New("Name", "Version", "Publisher", "OsType")
	for i := 0; i < len(*pluginInfoList); i++ {
		pluginInfo := (*pluginInfoList)[i]
		tbl.AddRow(pluginInfo.Name, pluginInfo.Version, pluginInfo.Publisher, pluginInfo.OsType)
	}
	tbl.Print()
	fmt.Println()
}

// get pluginInfo by name from online
func getPackageInfo(pluginName, version string) ([]pluginInfo, error) {
	postValue := struct {
		OsType     string `json:"osType"`
		PluginName string `json:"pluginName"`
		Version    string `json:"version"`
	}{
		OsType:     "linux",
		PluginName: pluginName,
		Version:    version,
	}
	listRet := ListRet{}
	if osutil.GetOsType() == osutil.OSWin {
		postValue.OsType = "windows"
	}
	postValueStr, err := marshal(&postValue)
	if err != nil {
		return listRet.PluginList, err
	}
	// http 请求尝试3次
	ret, err := util.HttpPost(util.GetPluginListService(), postValueStr, "json")
	if err != nil {
		retry := 2
		for ; retry > 0 && err != nil; {
			retry--
			ret, err = util.HttpPost(util.GetPluginListService(), postValueStr, "json")
		}
	}
	if err != nil {
		return listRet.PluginList, err
	}
	if err := unmarshal(ret, &listRet); err != nil {
		return nil, err
	}
	return listRet.PluginList, nil
}

func getLocalPluginInfo(packageName string) (*pluginInfo, error) {
	installedPlugins, err := loadInstalledPlugins()
	if err != nil {
		return nil, err
	}
	for _, plugin := range installedPlugins {
		if plugin.Name == packageName {
			return &plugin, nil
		}
	}
	return nil, nil
}

func getOnlinePluginInfo(packageName, version string) (*pluginInfo, error) {
	pluginList, err := getPackageInfo(packageName, version)
	if err != nil {
		return nil, err
	}
	for _, plugin := range pluginList {
		if plugin.Name == packageName {
			return &plugin, nil
		}
	}
	return nil, nil
}

func (pm *PluginManager) List(pluginName string, local bool) error {
	var pluginInfoList []pluginInfo
	var err error
	if local {
		pluginInfoList, err = loadInstalledPlugins()
	} else {
		pluginInfoList, err = getPackageInfo(pluginName, "")
	}
	if err == nil {
		printPluginInfo(&pluginInfoList)
	} else {
		if local {
			fmt.Print("List " + LOAD_INSTALLEDPLUGINS_ERR_STR + "Load installed_plugins err: " + err.Error())
		} else {
			fmt.Print("List " + GET_ONLINE_PACKAGE_INFO_ERR_STR + "Get plugin info from online err: " + err.Error())
		}
	}
	return err
}

func (pm *PluginManager) ShowPluginStatus() error {
	log.GetLogger().Infoln("Enter showPluginStatus")
	installedPlugins, err := loadInstalledPlugins()
	if err != nil {
		fmt.Print("ShowPluginStatus " + LOAD_INSTALLEDPLUGINS_ERR_STR + "Load installed_plugins err: " + err.Error())
		return err
	}
	log.GetLogger().Infof("Count of installed plugins: %d", len(installedPlugins))
	timeout := 10
	exitCode := 0
	statusList := []pluginStatus{}
	pluginPath := PLUGINDIR + Separator
	args := []string{"--status"}
	for _, plugin := range installedPlugins {
		if plugin.PluginType == PLUGIN_PERSIST {
			cmd := pluginPath + plugin.Name + Separator + plugin.Version + Separator + plugin.RunPath
			processCmd := process.NewProcessCmd()
			log.GetLogger().Info(fmt.Sprintf("cmdPath: %s, params: %+q\n", cmd, args))
			exitCode, _, err = processCmd.SyncRun("", cmd, args, nil, nil, nil, nil, timeout)
			status := pluginStatus{
				PluginId: plugin.PluginId,
				Name:     plugin.Name,
				OsType:   plugin.OsType,
				Arch:     plugin.Arch,
				Version:  plugin.Version,
				Status:   PERSIST_FAIL,
			}
			if exitCode == 0 {
				status.Status = PERSIST_RUNNING
			}
			statusList = append(statusList, status)
			exitCode = 0
		}
	}
	if len(statusList) > 0 {
		jsonStr := "["
		for _, status := range statusList {
			s, err := marshal(&status)
			if err != nil {
				return err
			}
			jsonStr += s
		}
		jsonStr += "]"
		fmt.Print(jsonStr)
	}
	return nil
}

func (pm *PluginManager) ExecutePlugin(file, pluginName, pluginId, params, separator, paramsV2, version string, local bool) (exitCode int, err error) {
	log.GetLogger().Infoln("Enter ExecutePlugin")
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

// run plugin from plugin_file.zip
func (pm *PluginManager) executePluginFromFile(file string, paramList []string, timeout int) (exitCode int, err error) {
	log.GetLogger().Infof("Enter executePluginFromFile")
	fmt.Println("Execute plugin from file: ", file)
	exitCode = SUCCESS
	if !util.CheckFileIsExist(file) {
		err = errors.New("File not exist: " + file)
		exitCode = PACKAGE_NOT_FOUND
		fmt.Print("ExecutePluginFromFile " + PACKAGE_NOT_FOUND_STR + "Package file not exist: " + file)
		return
	}
	cmdPath := ""
	idx := strings.LastIndex(file, Separator)
	pluginName := file
	dirName := "."
	if idx > 0 {
		pluginName = file[idx+1:]
		dirName = file[:idx]
	}
	idx = strings.Index(pluginName, ".zip")
	if idx <= 0 {
		err = errors.New("Package file not a zip file: " + file)
		exitCode = PACKAGE_FORMART_ERR
		fmt.Print("ExecutePluginFromFile " + PACKAGE_FORMART_ERR_STR + "Package file isn`t a zip file: " + file)
		return
	}
	pluginName = pluginName[:idx]
	dirName = dirName + Separator + pluginName
	util.MakeSurePath(dirName)
	if pm.Verbose {
		fmt.Printf("Unzip to %s ...\n", dirName)
	}
	unzipdir := dirName
	if err = util.Unzip(file, unzipdir); err != nil {
		exitCode = UNZIP_ERR
		tip := fmt.Sprintf("Unzip err, file is [%s], target dir is [%s], err is [%s]", file, unzipdir, err.Error())
		fmt.Print("ExecutePluginFromFile " + UNZIP_ERR_STR + tip)
		return
	}
	config_path := dirName + Separator + "config.json"
	if !util.CheckFileIsExist(config_path) {
		config_path = dirName + Separator + pluginName + Separator + "config.json"
		if !util.CheckFileIsExist(config_path) {
			exitCode = PLUGIN_FORMAT_ERR
			fmt.Print("ExecutePluginFromFile " + PLUGIN_FORMAT_ERR_STR + "File config.json not exist.")
			return
		}
		dirName = dirName + Separator + pluginName
	}
	config := pluginConfig{}
	var content []byte
	if content, err = unmarshalFile(config_path, &config); err != nil {
		exitCode = UNMARSHAL_ERR
		tip := fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", string(content), err.Error())
		fmt.Print("ExecutePluginFromFile " + UNMARSHAL_ERR_STR + tip)
		return
	}
	if config.Name == "" {
		err = errors.New("Plugin name in config is empty")
		exitCode = PLUGIN_FORMAT_ERR
		fmt.Print("ExecutePluginFromFile " + PLUGIN_FORMAT_ERR_STR + "The config.Name is empty.")
		return
	}
	if config.OsType != osutil.GetOsType() {
		err = errors.New("Plugin ostype not suit for this system")
		exitCode = PLUGIN_FORMAT_ERR
		tip := fmt.Sprintf("Plugin ostype[%s] not suit for this system[%s]\n", config.OsType, osutil.GetOsType())
		fmt.Print("ExecutePluginFromFile " + PLUGIN_FORMAT_ERR_STR + tip)
		return
	}
	var installedPlugins []pluginInfo
	installedPlugins, err = loadInstalledPlugins()
	if err != nil {
		exitCode = LOAD_INSTALLEDPLUGINS_ERR
		fmt.Print("ExecutePluginFromFile " + LOAD_INSTALLEDPLUGINS_ERR_STR + "Load installed_plugins err: " + err.Error())
		return
	}
	var plugin *pluginInfo
	pluginIndex := -1
	for idx, plugininfo := range installedPlugins {
		if plugininfo.Name == config.Name {
			plugin = &plugininfo
			pluginIndex = idx
			break
		}
	}
	if plugin != nil {
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
		plugin = &pluginInfo{
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
	plugin.OsType = config.OsType
	plugin.RunPath = config.RunPath
	plugin.Timeout = config.Timeout
	plugin.Publisher = config.Publisher
	plugin.Version = config.Version
	plugin.PluginType = config.PluginType
	plugin.Url = "local"
	var md5Str string
	md5Str, err = util.ComputeMd5(file)
	if err != nil {
		exitCode = MD5_CHECK_FAIL
		fmt.Print("ExecutePluginFromFile " + MD5_CHECK_FAIL_STR + "Compute md5 of plugin file err: " + err.Error())
		return
	}
	plugin.Md5 = md5Str

	pluginPath := PLUGINDIR + Separator
	pluginPath = pluginPath + plugin.Name + Separator + plugin.Version + Separator
	util.MakeSurePath(pluginPath)
	util.CopyDir(dirName, pluginPath)
	cmdPath = pluginPath + config.RunPath
	if osutil.GetOsType() != osutil.OSWin {
		if err = exec.Command("chmod", "744", cmdPath).Run(); err != nil {
			exitCode = EXECUTABLE_PERMISSION_ERR
			fmt.Print("ExecutePluginFromFile " + EXECUTABLE_PERMISSION_ERR_STR + "Make plugin file executable err: " + err.Error())
			return
		}
	}
	if pluginIndex == -1 {
		plugin.PluginId = "local_" + plugin.Name + "_" + plugin.Version
		installedPlugins = append(installedPlugins, *plugin)
	} else {
		installedPlugins[pluginIndex] = *plugin
	}
	if err = dumpInstalledPlugins(installedPlugins); err != nil {
		exitCode = DUMP_INSTALLEDPLUGINS_ERR
		fmt.Print("ExecutePluginFromFile " + DUMP_INSTALLEDPLUGINS_ERR_STR + "Upload installed_plugins file err: " + err.Error())
		return
	}
	fmt.Printf("Plugin[%s] installed!\n", plugin.Name)
	os.RemoveAll(unzipdir)

	return pm.executePlugin(cmdPath, paramList, timeout)
}

func (pm *PluginManager) executePluginOnlineOrLocal(pluginName string, pluginId string, version string, paramList []string, timeout int, local bool) (exitCode int, err error) {
	log.GetLogger().Info("Enter executePluginOnlineOrLocal")
	useLocal := false
	cmdPath := ""
	if !local {
		// didn't set --local, so local & online both try
		var localInfo *pluginInfo = nil
		var onlineInfo *pluginInfo = nil
		localInfo, err = getLocalPluginInfo(pluginName)
		if err != nil {
			exitCode = LOAD_INSTALLEDPLUGINS_ERR
			fmt.Print("ExecutePluginOnlineOrLocal " + LOAD_INSTALLEDPLUGINS_ERR_STR + "Load installed_plugins err: " + err.Error())
			return
		}
		onlineInfo, err = getOnlinePluginInfo(pluginName, version)
		if err != nil {
			exitCode = GET_ONLINE_PACKAGE_INFO_ERR
			fmt.Print("ExecutePluginOnlineOrLocal " + GET_ONLINE_PACKAGE_INFO_ERR_STR + "Get plugin info from online err: " + err.Error())
			return
		}
		if localInfo != nil {
			if onlineInfo != nil {
				if versionutil.CompareVersion(localInfo.Version, onlineInfo.Version) == 0 {
					useLocal = true
				}
			} else {
				useLocal = true
			}
		} else {
			if onlineInfo == nil {
				exitCode = PACKAGE_NOT_FOUND
				tip := fmt.Sprintf("Could not found package [%s] version[%s]", pluginName, version)
				fmt.Print("ExecutePluginOnlineOrLocal " + PACKAGE_NOT_FOUND_STR + tip)
				return
			}
		}
		if useLocal {
			pluginPath := PLUGINDIR + Separator
			pluginPath = pluginPath + localInfo.Name + Separator + localInfo.Version + Separator
			cmdPath = pluginPath + localInfo.RunPath
		} else {
			// pull package
			filePath := PLUGINDIR + Separator + pluginName + ".zip"
			log.GetLogger().Infoln("Downloading package from ", onlineInfo.Url)
			if err = util.HttpDownlod(onlineInfo.Url, filePath); err != nil {
				retry := 2
				for ; retry > 0 && err != nil; {
					retry--
					err = util.HttpDownlod(onlineInfo.Url, filePath)
				}
				if err != nil {
					exitCode = DOWNLOAD_FAIL
					tip := fmt.Sprintf("Downloading package failed, url is [%s], err is [%s]", onlineInfo.Url, err.Error())
					fmt.Print("ExecutePluginOnlineOrLocal " + DOWNLOAD_FAIL_STR + tip)
					return
				}
			}
			log.GetLogger().Infoln("Check MD5...")
			md5Str := ""
			md5Str, err = util.ComputeMd5(filePath)
			if err != nil {
				exitCode = MD5_CHECK_FAIL
				fmt.Print("ExecutePluginOnlineOrLocal " + MD5_CHECK_FAIL_STR + "Compute md5 of plugin file err: " + err.Error())
				return
			}
			if strings.ToLower(md5Str) != strings.ToLower(onlineInfo.Md5) {
				log.GetLogger().Errorf("Md5 not match, onlineInfo.Md5[%s], package file md5[%s]\n", onlineInfo.Md5, md5Str)
				err = errors.New("Md5 not macth")
				exitCode = MD5_CHECK_FAIL
				tip := fmt.Sprintf("Md5 not match, onlineInfo.Md5 is [%s], real md5 is [%s]", onlineInfo.Md5, md5Str)
				fmt.Print("ExecutePluginOnlineOrLocal " + MD5_CHECK_FAIL_STR + tip)
				return
			}
			unzipdir := PLUGINDIR + Separator + onlineInfo.Name + Separator + onlineInfo.Version
			util.MakeSurePath(unzipdir)
			log.GetLogger().Infoln("Unzip package...")
			if err = util.Unzip(filePath, unzipdir); err != nil {
				exitCode = UNZIP_ERR
				fmt.Print("ExecutePluginOnlineOrLocal " + UNZIP_ERR_STR + "Unzip package err: ", err.Error())
				return
			}
			os.RemoveAll(filePath)
			cmdPath = unzipdir + Separator + onlineInfo.RunPath
			if osutil.GetOsType() != osutil.OSWin {
				if err = exec.Command("chmod", "744", cmdPath).Run(); err != nil {
					exitCode = EXECUTABLE_PERMISSION_ERR
					fmt.Print("ExecutePluginOnlineOrLocal " + EXECUTABLE_PERMISSION_ERR_STR + "Make plugin file executable err: " + err.Error())
					return
				}
			}
			// update INSTALLEDPLUGINS file
			var installedPlugins []pluginInfo
			installedPlugins, err = loadInstalledPlugins()
			if err != nil {
				exitCode = LOAD_INSTALLEDPLUGINS_ERR
				fmt.Print("ExecutePluginOnlineOrLocal " + LOAD_INSTALLEDPLUGINS_ERR_STR + "Load installed_plugins err: " + err.Error())
				return
			}
			pluginIndex := -1
			for idx, plugininfo := range installedPlugins {
				if plugininfo.Name == onlineInfo.Name {
					pluginIndex = idx
					break
				}
			}
			if pluginIndex == -1 {
				installedPlugins = append(installedPlugins, *onlineInfo)
			} else {
				installedPlugins[pluginIndex] = *onlineInfo
			}
			err = dumpInstalledPlugins(installedPlugins)
			if err != nil {
				exitCode = DUMP_INSTALLEDPLUGINS_ERR
				fmt.Print("ExecutePluginOnlineOrLocal " + DUMP_INSTALLEDPLUGINS_ERR_STR + "Upload installed_plugins file err: " + err.Error())
				return
			}
		}
	} else {
		// execute local plugin
		var localInfo *pluginInfo
		localInfo, err = getLocalPluginInfo(pluginName)
		if err != nil {
			exitCode = LOAD_INSTALLEDPLUGINS_ERR
			fmt.Print("ExecutePluginOnlineOrLocal " + LOAD_INSTALLEDPLUGINS_ERR_STR + "Load installed_plugins err: " + err.Error())
			return
		} else if localInfo == nil {
			tip := fmt.Sprintf("Could not found local package [%s]", pluginName)
			err = errors.New("Could not found package")
			exitCode = PACKAGE_NOT_FOUND
			fmt.Print("ExecutePluginOnlineOrLocal " + PACKAGE_NOT_FOUND_STR + tip)
			return
		}
		cmdPath = PLUGINDIR + Separator + localInfo.Name + Separator + localInfo.Version + Separator + localInfo.RunPath
	}
	return pm.executePlugin(cmdPath, paramList, timeout)
}

func (pm *PluginManager) executePlugin(cmdPath string, paramList []string, timeout int) (exitCode int, err error) {
	log.GetLogger().Infof("Enter executePlugin, cmdPath[%s] paramList[%v] paramCount[%d] timeout[%d]\n", cmdPath, paramList, len(paramList), timeout)
	if !util.CheckFileIsExist(cmdPath) {
		log.GetLogger().Infoln("Cmd file not exist: ", cmdPath)
		err = errors.New("Cmd file not exist: " + cmdPath)
		exitCode = PACKAGE_FORMART_ERR
		fmt.Print("ExecutePlugin " + PACKAGE_FORMART_ERR_STR + "Executable file not exist.")
		return
	}
	if pm.Verbose {
		fmt.Printf("Run cmd: %s, params: %v\n", cmdPath, paramList)
	}

	processCmd := process.NewProcessCmd()
	log.GetLogger().Info(fmt.Sprintf("cmdPath: %s, params: %+q\n", cmdPath, paramList))
	exitCode, _, err = processCmd.SyncRun("", cmdPath, paramList, os.Stdout, os.Stderr, os.Stdin, nil, timeout)
	return
}

func (pm *PluginManager) VerifyPlugin(url, params, separator, paramsV2 string) (exitCode int, err error) {
	log.GetLogger().Infof("Enter VerufyPlugin url[%s] params[%s] separator[%s]\n", url, params, separator)
	var paramList []string
	timeout := 60
	cmdPath := ""
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
	if len(url)>4 && url[:4] == "http" {
		if err = util.HttpDownlod(url, filePath); err != nil {
			exitCode = DOWNLOAD_FAIL
			tip := fmt.Sprintf("Downloading package failed, url is [%s], err is [%s]", url, err.Error())
			fmt.Print("VerifyPlugin " + DOWNLOAD_FAIL_STR + tip)
			return
		}
	} else {
		if err = FileProtocolDownload(url, filePath); err != nil {
			exitCode = DOWNLOAD_FAIL
			tip := fmt.Sprintf("Downloading package failed, url is [%s], err is [%s]", url, err.Error())
			fmt.Print("VerifyPlugin " + DOWNLOAD_FAIL_STR + tip)
			return
		}
	}

	unzipdir := PLUGINDIR + Separator + "verify_plugin_test"
	util.MakeSurePath(unzipdir)
	log.GetLogger().Infoln("Unzip package...")
	if err = util.Unzip(filePath, unzipdir); err != nil {
		exitCode = UNZIP_ERR
		fmt.Print("VerifyPlugin " + UNZIP_ERR_STR + "Unzip package err: ", err.Error())
		return
	}
	os.RemoveAll(filePath)

	configPath := unzipdir + Separator + "config.json"
	if !util.CheckFileIsExist(configPath) {
		err = errors.New("Can not find the config.json")
		exitCode = PLUGIN_FORMAT_ERR
		fmt.Print("VerifyPlugin " + PLUGIN_FORMAT_ERR_STR + "File config.json not exist.")
		return
	}
	config := pluginConfig{}
	var content []byte
	if content, err = unmarshalFile(configPath, &config); err != nil {
		exitCode = UNMARSHAL_ERR
		tip := fmt.Sprintf("Unmarshal config.json err, config.json is [%s], err is [%s]", string(content), err.Error())
		fmt.Print("VerifyPlugin " + UNMARSHAL_ERR_STR + tip)
		return
	}

	runPath := config.RunPath
	timeoutStr := config.Timeout
	cmdPath = unzipdir + Separator + runPath
	if !util.CheckFileIsExist(cmdPath) {
		err = errors.New("Can not find the cmd file")
		exitCode = PACKAGE_FORMART_ERR
		fmt.Print("VerifyPlugin " + PACKAGE_FORMART_ERR_STR + "Executable file not exist.")
		return
	}
	if osutil.GetOsType() != osutil.OSWin {
		err = exec.Command("chmod", "744", cmdPath).Run()
		if err != nil {
			exitCode = EXECUTABLE_PERMISSION_ERR
			fmt.Print("VerifyPlugin " + EXECUTABLE_PERMISSION_ERR_STR + "Make plugin file executable err: " + err.Error())
			return
		}
	}
	timeout = 60
	if t, err := strconv.Atoi(timeoutStr); err != nil {
		fmt.Println("config.Timeout is invalid: ", config.Timeout)
	} else {
		timeout = t
	}

	return pm.executePlugin(cmdPath, paramList, timeout)
}
