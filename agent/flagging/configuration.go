package flagging

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/common/fileutil"
	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/metaserver"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// Config item names
const (
	UPDATOR_BOOTSTRAP_UPDATE      = "updator.bootstrapUpdate"
	UPDATOR_UPDATE                = "updator.update"
	TASK_CONCURRENCY_HARDLIMIT    = "task.concurrencyHardLimit"
	TASK_KEEP_SCRIPT_FILE         = "task.keepScriptFile"
	TASK_PROTOCOL                 = "task.protocol"
	RESOURCE_CPU_LIMIT            = "resource.cpuLimit"
	RESOURCE_MEM_LIMIT            = "resource.memLimit"
	RESOURCE_OVERLOAD_LIMIT       = "resource.overloadLimit"
	RESOURCE_LOG_FILE_COUNT_LIMIT = "resource.logFileCountLimit"
	RESOURCE_LOG_SIZE_LIMIT       = "resource.logSizeLimit"
	APISERVER_TRY_PRESETS         = "apiserver.tryPreset"
	ASSIST_DAEMON_ACTIVE          = "assistDaemon.active"
)

// Config item default values
const (
	DEFAULT_UPDATOR_BOOTSTRAP_UPDATE      = true
	DEFAULT_UPDATOR_UPDATE                = true
	DEFAULT_TASK_CONCURRENCY_HARDLIMIT    = int64(500)
	DEFAULT_TASK_KEEP_SCRIPT_FILE         = false
	DEFAULT_RESOURCE_CPU_LIMIT            = float64(20.0)           // %
	DEFAULT_RESOURCE_MEM_LIMIT            = int64(50 * 1024 * 1024) // Byte
	DEFAULT_RESOURCE_OVERLOAD_LIMIT       = int64(3)
	DEFAULT_RESOURCE_LOG_FILE_COUNT_LIMIT = int64(30)
	DEFAULT_RESOURCE_LOG_SIZE_LIMIT       = int64(100 * 1024 * 1024) // Byte
	DEFAULT_APISERVER_TRY_PRESETS         = true
	DEFAULT_ASSIST_DAEMON_ACTIVE          = false
)

const (
	MIN_RESOURCE_CPU_LIMIT            = float64(10.0)             // %
	MAX_RESOURCE_CPU_LIMIT            = float64(95.0)             // %
	MIN_RESOURCE_MEM_LIMIT            = int64(35 * 1024 * 1024)   // Byte
	MAX_RESOURCE_MEM_LIMIT            = int64(1024 * 1024 * 1024) // Byte
	MIN_RESOURCE_OVERLOAD_LIMIT       = int64(3)
	MAX_RESOURCE_OVERLOAD_LIMIT       = int64(math.MaxInt32) //compatible with 32-bit system
	MIN_TASK_CONCURRENCY_HARDLIMIT    = int64(1)
	MAX_TASK_CONCURRENCY_HARDLIMIT    = int64(math.MaxInt32) //compatible with 32-bit system
	MIN_RESOURCE_LOG_FILE_COUNT_LIMIT = int64(7)
	MAX_RESOURCE_LOG_FILE_COUNT_LIMIT = int64(365)
	MIN_RESOURCE_LOG_SIZE_LIMIT       = int64(10 * 1024 * 1024)   // Byte
	MAX_RESOURCE_LOG_SIZE_LIMIT       = int64(1024 * 1024 * 1024) // Byte
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

const (
	PriorityDefault    = 0
	PriorityApi        = 1
	PriorityMetaServer = 2
	PriorityConfigFile = 3
	PriorityRuntime    = 4
)

type intVar struct {
	value    int64
	min      int64
	max      int64
	priority int
}
type floatVar struct {
	value    float64
	min      float64
	max      float64
	priority int
}
type boolVar struct {
	value    bool
	priority int
}
type intVarWithSizeUnit struct {
	value    int64
	min      int64
	max      int64
	valueStr string
	priority int
}

type Value interface {
	Get() any
	GetString() string
	ParseAndSet(s string, set bool, priority int) error
	Copy() Value
	GetPriority() int
}

func newIntVar(v, min, max int64) *intVar       { return &intVar{value: v, min: min, max: max} }
func newFloatVar(v, min, max float64) *floatVar { return &floatVar{value: v, min: min, max: max} }
func newBoolVar(v bool) *boolVar                { return &boolVar{value: v} }
func newIntVarWithSizeUnit(v, min, max int64) *intVarWithSizeUnit {
	return &intVarWithSizeUnit{value: v, min: min, max: max, valueStr: fmt.Sprint(v)}
}
func (v *intVar) Get() any             { return v.value }
func (v *floatVar) Get() any           { return v.value }
func (v *boolVar) Get() any            { return v.value }
func (v *intVarWithSizeUnit) Get() any { return v.value }
func (v *intVar) GetString() string    { return fmt.Sprint(v.value) }
func (v *floatVar) GetString() string  { return fmt.Sprint(v.value) }
func (v *boolVar) GetString() string {
	if v.value {
		return VALUE_ENABLED
	}
	return VALUE_DISABLED
}
func (v *intVarWithSizeUnit) GetString() string { return fmt.Sprint(v.valueStr) }
func (v *intVar) ParseAndSet(s string, set bool, priority int) error {
	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		return err
	}
	if n > v.max || n < v.min {
		return errConfigValueOutOfRange
	}
	if set {
		v.value = n
		v.priority = priority
	}
	return nil
}
func (v *floatVar) ParseAndSet(s string, set bool, priority int) error {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}
	if n > v.max || n < v.min {
		return errConfigValueOutOfRange
	}
	if set {
		v.value = n
		v.priority = priority
	}
	return nil
}
func (v *boolVar) ParseAndSet(s string, set bool, priority int) error {
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
		v.priority = priority
	}
	return nil
}
func (v *intVarWithSizeUnit) ParseAndSet(s string, set bool, priority int) error {
	re := regexp.MustCompile(`^(\d+)\s*([KMG]?B?)$`)
	matches := re.FindStringSubmatch(strings.TrimSpace(s))
	if len(matches) != 3 {
		return errInvalidSizeUnit
	}

	valueStr := matches[1]
	unit := strings.ToUpper(strings.TrimSpace(matches[2]))

	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		return err
	}
	var sizeInt int64
	switch unit {
	case "B", "":
		sizeInt = value
	case "KB":
		sizeInt = value * 1024
	case "MB":
		sizeInt = value * 1024 * 1024
	case "GB":
		sizeInt = value * 1024 * 1024 * 1024
	}
	if sizeInt > v.max || sizeInt < v.min {
		return errConfigValueOutOfRange
	}
	if set {
		v.value = sizeInt
		v.priority = priority
		v.valueStr = s
	}
	return nil
}
func (v *intVar) GetPriority() int             { return v.priority }
func (v *floatVar) GetPriority() int           { return v.priority }
func (v *boolVar) GetPriority() int            { return v.priority }
func (v *intVarWithSizeUnit) GetPriority() int { return v.priority }

func (v *intVar) Copy() Value {
	return &intVar{value: v.value, min: v.min, max: v.max, priority: v.priority}
}
func (v *floatVar) Copy() Value {
	return &floatVar{value: v.value, min: v.min, max: v.max, priority: v.priority}
}
func (v *boolVar) Copy() Value { return &boolVar{value: v.value, priority: v.priority} }
func (v *intVarWithSizeUnit) Copy() Value {
	return &intVarWithSizeUnit{value: v.value, min: v.min, max: v.max, priority: v.priority, valueStr: v.valueStr}
}
func (v *intVar) GetValue() int64             { return v.value }
func (v *floatVar) GetValue() float64         { return v.value }
func (v *boolVar) GetValue() bool             { return v.value }
func (v *intVarWithSizeUnit) GetValue() int64 { return v.value }

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

	_oneTimeTaskConfigs = []string{TASK_PROTOCOL} //not in _runtimeConfig
)

var (
	errUnknownConfigName     = errors.New("unknown config name")
	errTypeNotMatch          = errors.New("type mismatch")
	errConfigValueOutOfRange = errors.New("config value is out of valid range")
	errInvalidSizeUnit       = errors.New("invalid size unit, should be empty or B/KB/MB")

	featureTypes  = []string{"task", "update", "resource"}
	jsonToFlagMap = map[string]string{
		"concurrencyHardLimit": TASK_CONCURRENCY_HARDLIMIT, // for task
		"keepScriptFile":       TASK_KEEP_SCRIPT_FILE,
		"protocol":             TASK_PROTOCOL,

		"bootstrapUpdate": UPDATOR_BOOTSTRAP_UPDATE, // for update
		"enableUpdate":    UPDATOR_UPDATE,

		"cpuLimit":          RESOURCE_CPU_LIMIT, // for resource
		"memLimit":          RESOURCE_MEM_LIMIT,
		"overloadLimit":     RESOURCE_OVERLOAD_LIMIT,
		"logFileCountLimit": RESOURCE_LOG_FILE_COUNT_LIMIT,
		"logSizeLimit":      RESOURCE_LOG_SIZE_LIMIT,
	}
)

var (
	_RUNTIME_UPDATOR_BOOTSTRAP_UPDATE      *boolVar
	_RUNTIME_UPDATOR_UPDATE                *boolVar
	_RUNTIME_TASK_CONCURRENCY_HARDLIMIT    *intVar
	_RUNTIME_TASK_KEEP_SCRIPT_FILE         *boolVar
	_RUNTIME_RESOURCE_CPU_LIMIT            *floatVar
	_RUNTIME_RESOURCE_MEM_LIMIT            *intVarWithSizeUnit
	_RUNTIME_RESOURCE_OVERLOAD_LIMIT       *intVar
	_RUNTIME_RESOURCE_LOG_FILE_COUNT_LIMIT *intVar
	_RUNTIME_RESOURCE_LOG_SIZE_LIMIT       *intVarWithSizeUnit
	_RUNTIME_APISERVER_TRY_PRESETS         *boolVar
	_RUNTIME_ASSIST_DAEMON_ACTIVE          *boolVar
)

func init() {
	_defaultConfig = make(agentConfig)
	_runtimeConfig = make(agentConfig)

	// _defaultConfig defines the data type and default value of each configuration item
	_defaultConfig[UPDATOR_BOOTSTRAP_UPDATE] = newBoolVar(DEFAULT_UPDATOR_BOOTSTRAP_UPDATE)
	_defaultConfig[UPDATOR_UPDATE] = newBoolVar(DEFAULT_UPDATOR_UPDATE)

	_defaultConfig[TASK_CONCURRENCY_HARDLIMIT] = newIntVar(DEFAULT_TASK_CONCURRENCY_HARDLIMIT, MIN_TASK_CONCURRENCY_HARDLIMIT, MAX_TASK_CONCURRENCY_HARDLIMIT)
	_defaultConfig[TASK_KEEP_SCRIPT_FILE] = newBoolVar(DEFAULT_TASK_KEEP_SCRIPT_FILE)

	_defaultConfig[RESOURCE_CPU_LIMIT] = newFloatVar(DEFAULT_RESOURCE_CPU_LIMIT, MIN_RESOURCE_CPU_LIMIT, MAX_RESOURCE_CPU_LIMIT)
	_defaultConfig[RESOURCE_MEM_LIMIT] = newIntVarWithSizeUnit(DEFAULT_RESOURCE_MEM_LIMIT, MIN_RESOURCE_MEM_LIMIT, MAX_RESOURCE_MEM_LIMIT)
	_defaultConfig[RESOURCE_OVERLOAD_LIMIT] = newIntVar(DEFAULT_RESOURCE_OVERLOAD_LIMIT, MIN_RESOURCE_OVERLOAD_LIMIT, MAX_RESOURCE_OVERLOAD_LIMIT)
	_defaultConfig[RESOURCE_LOG_FILE_COUNT_LIMIT] = newIntVar(DEFAULT_RESOURCE_LOG_FILE_COUNT_LIMIT, MIN_RESOURCE_LOG_FILE_COUNT_LIMIT, MAX_RESOURCE_LOG_FILE_COUNT_LIMIT)
	_defaultConfig[RESOURCE_LOG_SIZE_LIMIT] = newIntVarWithSizeUnit(DEFAULT_RESOURCE_LOG_SIZE_LIMIT, MIN_RESOURCE_LOG_SIZE_LIMIT, MAX_RESOURCE_LOG_SIZE_LIMIT)

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
	_RUNTIME_RESOURCE_MEM_LIMIT = _runtimeConfig[RESOURCE_MEM_LIMIT].(*intVarWithSizeUnit)
	_RUNTIME_RESOURCE_OVERLOAD_LIMIT = _runtimeConfig[RESOURCE_OVERLOAD_LIMIT].(*intVar)
	_RUNTIME_RESOURCE_LOG_FILE_COUNT_LIMIT = _runtimeConfig[RESOURCE_LOG_FILE_COUNT_LIMIT].(*intVar)
	_RUNTIME_RESOURCE_LOG_SIZE_LIMIT = _runtimeConfig[RESOURCE_LOG_SIZE_LIMIT].(*intVarWithSizeUnit)
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

	// Register callbacks for one-time execution task, only register and not apply
	for _, n := range _oneTimeTaskConfigs {
		if f, ok := callbacks[n]; ok {
			_callback[n] = f     //register
			delete(callbacks, n) //delete key in input callbacks
		}
	}

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
		} else if err := v.ParseAndSet(value, false, PriorityRuntime); err != nil {
			return err
		}
	}

	callbackKeys := []string{}
	for i := 0; i < len(kvList); i += 2 {
		property := kvList[i]
		value := kvList[i+1]
		_runtimeConfig[property].ParseAndSet(value, true, PriorityRuntime)
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
		} else if err := v.ParseAndSet(value, false, PriorityConfigFile); err != nil {
			return err
		}
		tempConf[property] = _defaultConfig[property].Copy()
		tempConf[property].ParseAndSet(value, true, PriorityConfigFile)
	}

	conf := loadConfigFile(logger, !crossVersion, crossVersion)
	for k, v := range tempConf {
		conf[k] = v
	}
	return dumpConfigFile(logger, conf, crossVersion)
}

// Update runtime config from api and call callback function to make configuration task effect.
func UpdateConfigApi(logger logrus.FieldLogger, toUpdate map[string]string) error {
	_runtimeConfigLock.Lock()
	defer _runtimeConfigLock.Unlock()
	_callbackLock.Lock()
	defer _callbackLock.Unlock()

	for _, key := range _oneTimeTaskConfigs {
		if value, ok := toUpdate[key]; ok {
			if f, exists := _callback[key]; exists {
				f(logger, value)
			}
			delete(toUpdate, key)
		}
	}

	for property, value := range toUpdate {
		if v, ok := _runtimeConfig[property]; !ok {
			return errUnknownConfigName
		} else {
			if _runtimeConfig[property].GetPriority() > PriorityApi { //not update config property upper than PriorityApi
				continue
			}
			if err := v.ParseAndSet(value, false, PriorityApi); err != nil {
				return err
			}
		}
	}

	callbackKeys := []string{}
	for property, value := range toUpdate {
		if _runtimeConfig[property].GetPriority() > PriorityApi { //not update config property upper than PriorityApi
			continue
		}
		_runtimeConfig[property].ParseAndSet(value, true, PriorityApi)
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

	callbackKeys := []string{}
	for k, v := range conf {
		if _, ok := _runtimeConfig[k]; !ok {
			_runtimeConfig[k] = v
			callbackKeys = append(callbackKeys, k)
			continue
		}

		if v.GetPriority() == PriorityDefault && _runtimeConfig[k].GetPriority() == PriorityApi { //only skip set value in this case
			continue
		}

		_runtimeConfig[k] = v
		callbackKeys = append(callbackKeys, k)
	}
	for _, property := range callbackKeys {
		if f, ok := _callback[property]; ok {
			f(logger, _runtimeConfig[property].Get())
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
		} else if err := v.ParseAndSet(value, false, PriorityConfigFile); err != nil {
			continue
		}
		conf[key] = _defaultConfig[key].Copy()
		conf[key].ParseAndSet(value, true, PriorityConfigFile)
	}
	return conf
}

// Parse config from heartbeat feature
func ParseConfigFromFeature(logger logrus.FieldLogger, content string) (items map[string]string, err error) {
	items = make(map[string]string)
	if !gjson.Valid(content) {
		logger.WithFields(log.Fields{"response": content}).Errorln("Invalid json response")
		return items, errors.New("invalid json response")
	}

	json := gjson.Parse(content)
	config := json.Get("config")
	if !config.Exists() || config.Type == gjson.Null {
		return
	}

	feature := config.Get("feature")
	if !feature.Exists() {
		logger.WithFields(log.Fields{"response": content}).Errorln("Invalid feature")
		return
	}

	for _, featureType := range featureTypes {
		arr := feature.Get(featureType)
		if !arr.Exists() || !arr.IsArray() {
			continue
		}

		for _, item := range arr.Array() {
			field := item.Get("field").String()
			if field == "" {
				continue
			}

			if flagKey, ok := jsonToFlagMap[field]; ok {
				value := item.Get("value")
				var strValue string
				switch {
				case value.Type == gjson.String, value.Type == gjson.Number:
					strValue = value.String()
				case value.Type == gjson.True, value.Type == gjson.False:
					if value.Bool() {
						strValue = VALUE_ENABLED
					} else {
						strValue = VALUE_DISABLED
					}
				default:
					logger.WithFields(logrus.Fields{"field": field, "type": value.Type}).Warn("Unsupported value type in heartbeat config")
					continue
				}
				items[flagKey] = strValue
			} else {
				logger.WithField("field", field).Debug("Field not mapped, skip")
			}
		}
	}
	return
}

func UpdateConfOnConfigReceived(logger logrus.FieldLogger, content string) (err error) {
	var featureItems = map[string]string{}
	featureItems, err = ParseConfigFromFeature(logger, content)
	if err != nil {
		logger.WithError(err).Errorln("ParseConfigFromFeature failed in heartbeat.")
	} else if len(featureItems) != 0 {
		logger.Infoln("ParseConfigFromFeature success in heartbeat: ", featureItems)
		err := UpdateConfigApi(logger, featureItems)
		if err != nil {
			logger.WithError(err).Errorln("UpdateConfigApi failed: ", featureItems)
		} else {
			logger.Infoln("UpdateConfigApi success.")
		}
	}
	return nil
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

func GetResourceLogFileCountLimit() int64 {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_LOG_FILE_COUNT_LIMIT.GetValue()
}

func GetResourceLogSizeLimit() int64 {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_LOG_SIZE_LIMIT.GetValue()
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

func GetUpdatorUpdatePriority() int {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_UPDATOR_UPDATE.GetPriority()
}

func GetTaskConcurrencyHardlimitPriority() int {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_TASK_CONCURRENCY_HARDLIMIT.GetPriority()
}

func GetTaskKeepScriptFilePriority() int {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_TASK_KEEP_SCRIPT_FILE.GetPriority()
}

func GetResourceCpuLimitPriority() int {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_CPU_LIMIT.GetPriority()
}

func GetResourceMemLimitPriority() int {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_MEM_LIMIT.GetPriority()
}

func GetResourceOverloadLimitPriority() int {
	_runtimeConfigLock.RLock()
	defer _runtimeConfigLock.RUnlock()

	return _RUNTIME_RESOURCE_OVERLOAD_LIMIT.GetPriority()
}
