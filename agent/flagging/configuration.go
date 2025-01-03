package flagging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/metaserver"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
)

// Config item names
const (
	UPDATOR_BOOTSTRAP_UPDATE   = "updator.bootstrapUpdate"
	UPDATOR_UPDATE             = "updator.update"
	TASK_CONCURRENCY_HARDLIMIT = "task.concurrencyHardLimit"
	TASK_KEEP_SCRIPT_FILE      = "task.keepScriptFile"
	RESOURCE_CPU_LIMIT         = "resource.cpuLimit"
	RESOURCE_MEM_LIMIT         = "resource.memLimit"
	RESOURCE_OVERLOAD_LIMIT    = "resource.overloadLimit"
	APISERVER_TRY_PRESETS      = "apiserver.tryPreset"
	ASSIST_DAEMON_ACTIVE       = "assistDaemon.active"
)

// Config item default values
const (
	DEFAULT_UPDATOR_BOOTSTRAP_UPDATE   = true
	DEFAULT_UPDATOR_UPDATE             = true
	DEFAULT_TASK_CONCURRENCY_HARDLIMIT = int64(500)
	DEFAULT_TASK_KEEP_SCRIPT_FILE      = false
	DEFAULT_RESOURCE_CPU_LIMIT         = float64(20.0)           // %
	DEFAULT_RESOURCE_MEM_LIMIT         = int64(50 * 1024 * 1024) // Byte
	DEFAULT_RESOURCE_OVERLOAD_LIMIT    = int64(3)
	DEFAULT_APISERVER_TRY_PRESETS      = true
	DEFAULT_ASSIST_DAEMON_ACTIVE       = true
)

const (
	configurationFile = "aliyun-assist.conf"

	VALUE_ENABLED  = "enabled"
	VALUE_DISABLED = "disabled"

	// The flag files of the old version which are used to control whether
	// to prohibit the auto-update
	disableUpdateFlagFilename          = "disable_update"
	disableBootstrapUpdateFlagFilename = "disable_bootstrap_update"
)

type intVar struct{ value int64 }
type floatVar struct{ value float64 }
type boolVar struct{ value bool }

type Value interface {
	Get() any
	GetString() string
	ParseAndSet(s string, set bool) error
	Copy() Value
}

func newIntVar(v int64) *intVar       { return &intVar{value: v} }
func newFloatVar(v float64) *floatVar { return &floatVar{value: v} }
func newBoolVar(v bool) *boolVar      { return &boolVar{value: v} }
func (v *intVar) Get() any            { return v.value }
func (v *floatVar) Get() any          { return v.value }
func (v *boolVar) Get() any           { return v.value }
func (v *intVar) GetString() string   { return fmt.Sprint(v.value) }
func (v *floatVar) GetString() string { return fmt.Sprint(v.value) }
func (v *boolVar) GetString() string {
	if v.value {
		return VALUE_ENABLED
	}
	return VALUE_DISABLED
}
func (v *intVar) ParseAndSet(s string, set bool) error {
	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return err
	}
	if set {
		v.value = n
	}
	return nil
}
func (v *floatVar) ParseAndSet(s string, set bool) error {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	if set {
		v.value = n
	}
	return nil
}
func (v *boolVar) ParseAndSet(s string, set bool) error {
	var n bool
	switch s {
	case VALUE_ENABLED:
		n = true
	case VALUE_DISABLED:
		n = false
	default:
		return errTypeNotMatch
	}
	if set {
		v.value = n
	}
	return nil
}

func (v *intVar) Copy() Value         { return &intVar{value: v.value} }
func (v *floatVar) Copy() Value       { return &floatVar{value: v.value} }
func (v *boolVar) Copy() Value        { return &boolVar{value: v.value} }
func (v *intVar) GetValue() int64     { return v.value }
func (v *floatVar) GetValue() float64 { return v.value }
func (v *boolVar) GetValue() bool     { return v.value }

type agentConfig map[string]Value
type Callback func(logger logrus.FieldLogger, value any)

var (
	// DO NOT MODIFY _defaultConfig
	// Note: The value of agentConfig is a reference type. Please use Value.Copy()
	// to copy it to avoid the reference being modified elsewhere.
	_defaultConfig agentConfig

	_runtimeConfig     agentConfig
	_runtimeConfigLock sync.RWMutex

	_callback     map[string]Callback
	_callbackLock sync.Mutex

	_configFileLock sync.Mutex
)

var (
	errUnknownConfigName = errors.New("unknown config name")
	errTypeNotMatch      = errors.New("type mismatch")
)

var (
	_RUNTIME_UPDATOR_BOOTSTRAP_UPDATE   *boolVar
	_RUNTIME_UPDATOR_UPDATE             *boolVar
	_RUNTIME_TASK_CONCURRENCY_HARDLIMIT *intVar
	_RUNTIME_TASK_KEEP_SCRIPT_FILE      *boolVar
	_RUNTIME_RESOURCE_CPU_LIMIT         *floatVar
	_RUNTIME_RESOURCE_MEM_LIMIT         *intVar
	_RUNTIME_RESOURCE_OVERLOAD_LIMIT    *intVar
	_RUNTIME_APISERVER_TRY_PRESETS      *boolVar
	_RUNTIME_ASSIST_DAEMON_ACTIVE       *boolVar
)

func init() {
	_defaultConfig = make(agentConfig)
	_runtimeConfig = make(agentConfig)

	// _defaultConfig defines the data type and default value of each configuration item
	_defaultConfig[UPDATOR_BOOTSTRAP_UPDATE] = newBoolVar(DEFAULT_UPDATOR_BOOTSTRAP_UPDATE)
	_defaultConfig[UPDATOR_UPDATE] = newBoolVar(DEFAULT_UPDATOR_UPDATE)

	_defaultConfig[TASK_CONCURRENCY_HARDLIMIT] = newIntVar(DEFAULT_TASK_CONCURRENCY_HARDLIMIT)
	_defaultConfig[TASK_KEEP_SCRIPT_FILE] = newBoolVar(DEFAULT_TASK_KEEP_SCRIPT_FILE)

	_defaultConfig[RESOURCE_CPU_LIMIT] = newFloatVar(DEFAULT_RESOURCE_CPU_LIMIT)
	_defaultConfig[RESOURCE_MEM_LIMIT] = newIntVar(DEFAULT_RESOURCE_MEM_LIMIT)
	_defaultConfig[RESOURCE_OVERLOAD_LIMIT] = newIntVar(DEFAULT_RESOURCE_OVERLOAD_LIMIT)

	_defaultConfig[APISERVER_TRY_PRESETS] = newBoolVar(DEFAULT_APISERVER_TRY_PRESETS)
	_defaultConfig[ASSIST_DAEMON_ACTIVE] = newBoolVar(DEFAULT_ASSIST_DAEMON_ACTIVE)
}

func InitConfig(logger logrus.FieldLogger) {
	_runtimeConfigLock.Lock()
	defer _runtimeConfigLock.Unlock()
	defer updateRuntime()

	_callbackLock.Lock()
	defer _callbackLock.Unlock()

	_runtimeConfig = loadAllConfig(logger)
	_callback = make(map[string]Callback)
}

func updateRuntime() {
	_RUNTIME_UPDATOR_BOOTSTRAP_UPDATE = _runtimeConfig[UPDATOR_BOOTSTRAP_UPDATE].(*boolVar)
	_RUNTIME_UPDATOR_UPDATE = _runtimeConfig[UPDATOR_UPDATE].(*boolVar)
	_RUNTIME_TASK_CONCURRENCY_HARDLIMIT = _runtimeConfig[TASK_CONCURRENCY_HARDLIMIT].(*intVar)
	_RUNTIME_TASK_KEEP_SCRIPT_FILE = _runtimeConfig[TASK_KEEP_SCRIPT_FILE].(*boolVar)
	_RUNTIME_RESOURCE_CPU_LIMIT = _runtimeConfig[RESOURCE_CPU_LIMIT].(*floatVar)
	_RUNTIME_RESOURCE_MEM_LIMIT = _runtimeConfig[RESOURCE_MEM_LIMIT].(*intVar)
	_RUNTIME_RESOURCE_OVERLOAD_LIMIT = _runtimeConfig[RESOURCE_OVERLOAD_LIMIT].(*intVar)
	_RUNTIME_APISERVER_TRY_PRESETS = _runtimeConfig[APISERVER_TRY_PRESETS].(*boolVar)
	_RUNTIME_ASSIST_DAEMON_ACTIVE = _runtimeConfig[ASSIST_DAEMON_ACTIVE].(*boolVar)
}

func GetAllConf(logger logrus.FieldLogger, runtime bool) map[string]string {
	res := make(map[string]string)
	if runtime {
		_runtimeConfigLock.RLock()
		defer _runtimeConfigLock.RUnlock()
		for k, v := range _runtimeConfig {
			res[k] = v.GetString()
		}
	} else {
		conf := loadAllConfig(logger)
		for k, v := range conf {
			res[k] = v.GetString()
		}
	}
	return res
}

// Register the callback function for each configuration item, and then call
// the callback function to make the configuration take effect.
func RegisterCallbackAndApply(logger logrus.FieldLogger, callbacks map[string]Callback) error {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()
	_callbackLock.Lock()
	defer _callbackLock.Unlock()

	for n, f := range callbacks {
		if _, ok := _runtimeConfig[n]; !ok {
			logger.Warning("Unknown configuration item: ", n)
		} else {
			_callback[n] = f
			f(logger, _runtimeConfig[n].Get())
		}
	}
	return nil
}

// Update runtime config and call callback function to make configuration task effect.
func UpdateConfigRuntime(logger logrus.FieldLogger, kvList []string) error {
	_runtimeConfigLock.Lock()
	defer _runtimeConfigLock.Unlock()
	_callbackLock.Lock()
	defer _callbackLock.Unlock()

	for i := 0; i < len(kvList); i += 2 {
		property := kvList[i]
		value := kvList[i+1]
		if v, ok := _runtimeConfig[property]; !ok {
			return errUnknownConfigName
		} else if err := v.ParseAndSet(value, false); err != nil {
			return err
		}
	}

	callbackKeys := []string{}
	for i := 0; i < len(kvList); i += 2 {
		property := kvList[i]
		value := kvList[i+1]
		_runtimeConfig[property].ParseAndSet(value, true)
		if _, ok := _callback[property]; ok {
			callbackKeys = append(callbackKeys, property)
		}
	}
	for _, property := range callbackKeys {
		if f, ok := _callback[property]; ok {
			f(logger, _runtimeConfig[property].Get())
		}

	}

	return nil
}

// Update configuration file.
func UpdateConfigFile(logger logrus.FieldLogger, kvList []string, crossVersion bool) error {
	tempConf := make(agentConfig)
	for i := 0; i < len(kvList); i += 2 {
		property := kvList[i]
		value := kvList[i+1]
		if v, ok := _defaultConfig[property]; !ok {
			return errUnknownConfigName
		} else if err := v.ParseAndSet(value, false); err != nil {
			return err
		}
		tempConf[property] = _defaultConfig[property].Copy()
		tempConf[property].ParseAndSet(value, true)
	}

	conf := loadConfigFile(logger, !crossVersion, crossVersion)
	for k, v := range tempConf {
		conf[k] = v
	}
	return dumpConfigFile(logger, conf, crossVersion)
}

// Reload all config
func ReloadConfig(logger logrus.FieldLogger) {
	conf := loadAllConfig(logger)
	if len(conf) == 0 {
		return
	}

	_runtimeConfigLock.Lock()
	defer _runtimeConfigLock.Unlock()
	defer updateRuntime()

	_callbackLock.Lock()
	defer _callbackLock.Unlock()

	for k, v := range conf {
		_runtimeConfig[k] = v
		// TODO: call callback function only if _runtimeConfig[k] changed
		if f, ok := _callback[k]; ok {
			f(logger, v)
		}
	}
}

func loadAllConfig(logger logrus.FieldLogger) agentConfig {
	conf := make(agentConfig)
	for k, v := range _defaultConfig {
		conf[k] = v.Copy()
	}

	// Load from metaserver
	confFromMetaserver := loadConfigFromMetaserver(logger)
	for k, v := range confFromMetaserver {
		conf[k] = v
	}
	// Load from config file to _runtimeConfig
	confFromFile := loadConfigFile(logger, true, true)
	for k, v := range confFromFile {
		conf[k] = v
	}
	return conf
}

// Read content from confPath and parse it to map[string]any.
func parseConfigFile(confPath string) (conf agentConfig, err error) {
	var content []byte
	content, err = os.ReadFile(confPath)
	if err != nil {
		return
	} else {
		conf = parseConfigContent(string(content))
	}
	return
}

func parseConfigContent(content string) agentConfig {
	conf := make(agentConfig)
	for _, line := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, "=")
		if idx == -1 || idx >= len(line)-1 {
			continue
		}
		key := line[:idx]
		value := line[idx+1:]
		if v, ok := _defaultConfig[key]; !ok {
			continue
		} else if err := v.ParseAndSet(value, false); err != nil {
			continue
		}
		conf[key] = _defaultConfig[key].Copy()
		conf[key].ParseAndSet(value, true)
	}
	return conf
}

// Load configuration from configuration file.
func loadConfigFile(logger logrus.FieldLogger, currentVer, crossVer bool) agentConfig {
	_configFileLock.Lock()
	defer _configFileLock.Unlock()

	var confCrossVer, confVer agentConfig
	if crossVer {
		confDir, err := pathutil.GetCrossVersionConfigPath()
		if err != nil {
			logger.WithError(err).Error("Get cross version config path failed.")
		} else {
			confCrossVer, err = parseConfigFile(filepath.Join(confDir, configurationFile))
			if err != nil {
				logger.WithError(err).Error("Parse cross version config file failed.")
			}
		}
	}

	if currentVer {
		confDir, err := pathutil.GetConfigPath()
		if err != nil {
			logger.WithError(err).Error("Get config path failed.")
		} else {
			confVer, err = parseConfigFile(filepath.Join(confDir, configurationFile))
			if err != nil {
				logger.WithError(err).Error("Parse config file failed.")
			}
		}
	}
	if confCrossVer == nil {
		confCrossVer = make(agentConfig)
	}

	// confVer has higher priority than confCrossVer
	for k, v := range confVer {
		confCrossVer[k] = v
	}

	// Item in legacy configuration will be ignored if there is an explicit configuration
	confLegacy := loadConfLegacy(currentVer, crossVer)
	for k, v := range confLegacy {
		if _, ok := confCrossVer[k]; !ok {
			confCrossVer[k] = v
		}
	}

	return confCrossVer
}

func dumpConfigFile(logger logrus.FieldLogger, conf agentConfig, crossVer bool) error {
	_configFileLock.Lock()
	defer _configFileLock.Unlock()

	var confDir string
	var err error
	if crossVer {
		confDir, err = pathutil.GetCrossVersionConfigPath()
	} else {
		confDir, err = pathutil.GetConfigPath()
	}
	if err != nil {
		logger.WithError(err).WithField("CrossVersion", crossVer).Error("Get config path failed.")
		return err
	}
	var res []string
	for k, v := range conf {
		res = append(res, fmt.Sprintf("%s=%s", k, v.GetString()))
	}
	content := strings.Join(res, "\n")
	confPath := filepath.Join(confDir, configurationFile)
	return os.WriteFile(confPath, []byte(content), 0644)
}

func loadConfLegacy(currentVer, crossVer bool) agentConfig {
	conf := make(agentConfig)
	if crossVer {
		if confDir, err := pathutil.GetCrossVersionConfigPath(); err == nil {
			if fileutil.CheckFileIsExist(filepath.Join(confDir, disableBootstrapUpdateFlagFilename)) {
				conf[UPDATOR_BOOTSTRAP_UPDATE] = newBoolVar(false)
			}
			if fileutil.CheckFileIsExist(filepath.Join(confDir, disableUpdateFlagFilename)) {
				conf[UPDATOR_UPDATE] = newBoolVar(false)
			}
		}
	}
	if currentVer {
		if confDir, err := pathutil.GetConfigPath(); err == nil {
			if fileutil.CheckFileIsExist(filepath.Join(confDir, disableBootstrapUpdateFlagFilename)) {
				conf[UPDATOR_BOOTSTRAP_UPDATE] = newBoolVar(false)
			}
			if fileutil.CheckFileIsExist(filepath.Join(confDir, disableUpdateFlagFilename)) {
				conf[UPDATOR_UPDATE] = newBoolVar(false)
			}
		}
	}

	return conf
}

// load configuration from metaserver api
func loadConfigFromMetaserver(logger logrus.FieldLogger) (conf agentConfig) {
	conf = make(agentConfig)

	content, err := metaserver.GetAgentProvision(logger, httpbase.WithTimeoutInSeconds(2))
	if err != nil {
		logger.WithError(err).Error("Get config from metaserver failed")
		return
	}
	logger.Info("Get config content from metaserver: ", content)
	conf = parseConfigContent(content)
	return
}

func GetUpdatorBootstrapUpdate() bool {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_UPDATOR_BOOTSTRAP_UPDATE.GetValue()
}

func GetUpdatorUpdate() bool {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_UPDATOR_UPDATE.GetValue()
}

func GetTaskConcurrencyHardlimit() int64 {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_TASK_CONCURRENCY_HARDLIMIT.GetValue()
}

func GetTaskKeepScriptFile() bool {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_TASK_KEEP_SCRIPT_FILE.GetValue()
}

func GetResourceCpuLimit() float64 {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_CPU_LIMIT.GetValue()
}

func GetResourceMemLimit() int64 {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_MEM_LIMIT.GetValue()
}

func GetResourceOverloadLimit() int64 {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_OVERLOAD_LIMIT.GetValue()
}

func GetApiserverTryPreset() bool {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_APISERVER_TRY_PRESETS.GetValue()
}

func GetAssistDaemonActive() bool {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_ASSIST_DAEMON_ACTIVE.GetValue()
}
