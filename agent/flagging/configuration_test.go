package flagging

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitConfig(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()

	var (
		updator_bootstrapUpdate bool
		updator_update          bool

		task_concurrencyHardLimit int64
		task_keepScriptFile       bool

		resource_cpuLimit      float64
		resource_memLimit      int64
		resource_overloadLimit int64

		resource_logCountLimit int64
		resource_logSizeLimit  int64

		apiserver_try_presets bool
		assistDaemon_active   bool
	)
	InitConfig(logrus.New())

	RegisterCallbackAndApply(logrus.New(), map[string]Callback{
		UPDATOR_BOOTSTRAP_UPDATE: func(logger logrus.FieldLogger, v any) { updator_bootstrapUpdate, _ = v.(bool) },
		UPDATOR_UPDATE:           func(logger logrus.FieldLogger, v any) { updator_update, _ = v.(bool) },

		TASK_CONCURRENCY_HARDLIMIT: func(logger logrus.FieldLogger, v any) { task_concurrencyHardLimit, _ = v.(int64) },
		TASK_KEEP_SCRIPT_FILE:      func(logger logrus.FieldLogger, v any) { task_keepScriptFile, _ = v.(bool) },

		RESOURCE_CPU_LIMIT:            func(logger logrus.FieldLogger, v any) { resource_cpuLimit, _ = v.(float64) },
		RESOURCE_MEM_LIMIT:            func(logger logrus.FieldLogger, v any) { resource_memLimit, _ = v.(int64) },
		RESOURCE_OVERLOAD_LIMIT:       func(logger logrus.FieldLogger, v any) { resource_overloadLimit, _ = v.(int64) },
		RESOURCE_LOG_FILE_COUNT_LIMIT: func(logger logrus.FieldLogger, v any) { resource_logCountLimit, _ = v.(int64) },
		RESOURCE_LOG_SIZE_LIMIT:       func(logger logrus.FieldLogger, v any) { resource_logSizeLimit, _ = v.(int64) },

		APISERVER_TRY_PRESETS: func(logger logrus.FieldLogger, v any) { apiserver_try_presets, _ = v.(bool) },
		ASSIST_DAEMON_ACTIVE:  func(logger logrus.FieldLogger, v any) { assistDaemon_active, _ = v.(bool) },
	})
	assert.Equal(t, DEFAULT_UPDATOR_BOOTSTRAP_UPDATE, updator_bootstrapUpdate)
	assert.Equal(t, DEFAULT_UPDATOR_UPDATE, updator_update)
	assert.Equal(t, int64(DEFAULT_TASK_CONCURRENCY_HARDLIMIT), task_concurrencyHardLimit)
	assert.Equal(t, DEFAULT_TASK_KEEP_SCRIPT_FILE, task_keepScriptFile)
	assert.Equal(t, DEFAULT_RESOURCE_CPU_LIMIT, resource_cpuLimit)
	assert.Equal(t, int64(DEFAULT_RESOURCE_MEM_LIMIT), resource_memLimit)
	assert.Equal(t, int64(DEFAULT_RESOURCE_OVERLOAD_LIMIT), resource_overloadLimit)
	assert.Equal(t, int64(DEFAULT_RESOURCE_LOG_FILE_COUNT_LIMIT), resource_logCountLimit)
	assert.Equal(t, int64(DEFAULT_RESOURCE_LOG_SIZE_LIMIT), resource_logSizeLimit)
	assert.Equal(t, DEFAULT_APISERVER_TRY_PRESETS, apiserver_try_presets)
	assert.Equal(t, DEFAULT_ASSIST_DAEMON_ACTIVE, assistDaemon_active)

	assert.Equal(t, DEFAULT_UPDATOR_BOOTSTRAP_UPDATE, GetUpdatorBootstrapUpdate())
	assert.Equal(t, DEFAULT_UPDATOR_UPDATE, GetUpdatorUpdate())
	assert.Equal(t, int64(DEFAULT_TASK_CONCURRENCY_HARDLIMIT), GetTaskConcurrencyHardlimit())
	assert.Equal(t, DEFAULT_TASK_KEEP_SCRIPT_FILE, GetTaskKeepScriptFile())
	assert.Equal(t, DEFAULT_RESOURCE_CPU_LIMIT, GetResourceCpuLimit())
	assert.Equal(t, int64(DEFAULT_RESOURCE_MEM_LIMIT), GetResourceMemLimit())
	assert.Equal(t, int64(DEFAULT_RESOURCE_OVERLOAD_LIMIT), GetResourceOverloadLimit())
	assert.Equal(t, int64(DEFAULT_RESOURCE_LOG_FILE_COUNT_LIMIT), GetResourceLogFileCountLimit())
	assert.Equal(t, int64(DEFAULT_RESOURCE_LOG_SIZE_LIMIT), GetResourceLogSizeLimit())
	assert.Equal(t, DEFAULT_APISERVER_TRY_PRESETS, GetApiserverTryPreset())
	assert.Equal(t, DEFAULT_ASSIST_DAEMON_ACTIVE, GetAssistDaemonActive())
}

func TestUpdateConfigRuntime(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()

	var (
		updator_bootstrapUpdate bool
		updator_update          bool

		task_concurrencyHardLimit int64
		task_keepScriptFile       bool

		resource_cpuLimit      float64
		resource_memLimit      int64
		resource_overloadLimit int64
		resource_logCountLimit int64
		resource_logSizeLimit  int64
	)

	InitConfig(logrus.New())
	RegisterCallbackAndApply(logrus.New(), map[string]Callback{
		UPDATOR_BOOTSTRAP_UPDATE: func(logger logrus.FieldLogger, v any) { updator_bootstrapUpdate, _ = v.(bool) },
		UPDATOR_UPDATE:           func(logger logrus.FieldLogger, v any) { updator_update, _ = v.(bool) },

		TASK_CONCURRENCY_HARDLIMIT: func(logger logrus.FieldLogger, v any) { task_concurrencyHardLimit, _ = v.(int64) },
		TASK_KEEP_SCRIPT_FILE:      func(logger logrus.FieldLogger, v any) { task_keepScriptFile, _ = v.(bool) },

		RESOURCE_CPU_LIMIT:            func(logger logrus.FieldLogger, v any) { resource_cpuLimit, _ = v.(float64) },
		RESOURCE_MEM_LIMIT:            func(logger logrus.FieldLogger, v any) { resource_memLimit, _ = v.(int64) },
		RESOURCE_OVERLOAD_LIMIT:       func(logger logrus.FieldLogger, v any) { resource_overloadLimit, _ = v.(int64) },
		RESOURCE_LOG_FILE_COUNT_LIMIT: func(logger logrus.FieldLogger, v any) { resource_logCountLimit, _ = v.(int64) },
		RESOURCE_LOG_SIZE_LIMIT:       func(logger logrus.FieldLogger, v any) { resource_logSizeLimit, _ = v.(int64) },
	})

	t.Run("valid updates", func(t *testing.T) {
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{UPDATOR_BOOTSTRAP_UPDATE, VALUE_DISABLED}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{UPDATOR_UPDATE, VALUE_DISABLED}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{TASK_CONCURRENCY_HARDLIMIT, "501"}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{TASK_KEEP_SCRIPT_FILE, VALUE_ENABLED}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_CPU_LIMIT, "21.1"}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_MEM_LIMIT, "40MB"}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_OVERLOAD_LIMIT, "4"}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_LOG_FILE_COUNT_LIMIT, "40"}))
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_LOG_SIZE_LIMIT, "41943040"}))
		assert.ErrorContains(t, UpdateConfigRuntime(logrus.New(), []string{"undifined", "4"}), errUnknownConfigName.Error())

		assert.Equal(t, false, updator_bootstrapUpdate)
		assert.Equal(t, false, updator_update)
		assert.Equal(t, int64(501), task_concurrencyHardLimit)
		assert.Equal(t, true, task_keepScriptFile)
		assert.Equal(t, float64(21.1), resource_cpuLimit)
		assert.Equal(t, int64(41943040), resource_memLimit)
		assert.Equal(t, int64(4), resource_overloadLimit)
		assert.Equal(t, int64(40), resource_logCountLimit)
		assert.Equal(t, int64(41943040), resource_logSizeLimit)
		assert.Equal(t, false, GetUpdatorBootstrapUpdate())
		assert.Equal(t, false, GetUpdatorUpdate())
		assert.Equal(t, int64(501), GetTaskConcurrencyHardlimit())
		assert.Equal(t, true, GetTaskKeepScriptFile())
		assert.Equal(t, float64(21.1), GetResourceCpuLimit())
		assert.Equal(t, int64(41943040), GetResourceMemLimit())
		assert.Equal(t, int64(4), GetResourceOverloadLimit())
		assert.Equal(t, int64(40), GetResourceLogFileCountLimit())
		assert.Equal(t, int64(41943040), GetResourceLogSizeLimit())
	})

	t.Run("boundary value tests", func(t *testing.T) {
		// CPU Limit
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_CPU_LIMIT, fmt.Sprintf("%.1f", MIN_RESOURCE_CPU_LIMIT)}))
		assert.InDelta(t, MIN_RESOURCE_CPU_LIMIT, GetResourceCpuLimit(), 0.001)
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_CPU_LIMIT, fmt.Sprintf("%.1f", MAX_RESOURCE_CPU_LIMIT)}))
		assert.InDelta(t, MAX_RESOURCE_CPU_LIMIT, GetResourceCpuLimit(), 0.001)

		// RESOURCE_MEM_LIMIT 边界
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_MEM_LIMIT, strconv.FormatInt(int64(MIN_RESOURCE_MEM_LIMIT), 10)}))
		assert.Equal(t, MIN_RESOURCE_MEM_LIMIT, GetResourceMemLimit())
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_MEM_LIMIT, strconv.FormatInt(MAX_RESOURCE_MEM_LIMIT, 10)}))
		assert.Equal(t, MAX_RESOURCE_MEM_LIMIT, GetResourceMemLimit())

		// Concurrency
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{TASK_CONCURRENCY_HARDLIMIT, strconv.FormatInt(MIN_TASK_CONCURRENCY_HARDLIMIT, 10)}))
		assert.Equal(t, MIN_TASK_CONCURRENCY_HARDLIMIT, GetTaskConcurrencyHardlimit())
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{TASK_CONCURRENCY_HARDLIMIT, strconv.FormatInt(MAX_TASK_CONCURRENCY_HARDLIMIT, 10)}))
		assert.Equal(t, MAX_TASK_CONCURRENCY_HARDLIMIT, GetTaskConcurrencyHardlimit())

		// Overload
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_OVERLOAD_LIMIT, strconv.FormatInt(MIN_RESOURCE_OVERLOAD_LIMIT, 10)}))
		assert.Equal(t, MIN_RESOURCE_OVERLOAD_LIMIT, GetResourceOverloadLimit())
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_OVERLOAD_LIMIT, strconv.FormatInt(MAX_RESOURCE_OVERLOAD_LIMIT, 10)}))
		assert.Equal(t, MAX_RESOURCE_OVERLOAD_LIMIT, GetResourceOverloadLimit())

		// Log Count
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_LOG_FILE_COUNT_LIMIT, strconv.FormatInt(MIN_RESOURCE_LOG_FILE_COUNT_LIMIT, 10)}))
		assert.Equal(t, MIN_RESOURCE_LOG_FILE_COUNT_LIMIT, GetResourceLogFileCountLimit())
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_LOG_FILE_COUNT_LIMIT, strconv.FormatInt(MAX_RESOURCE_LOG_FILE_COUNT_LIMIT, 10)}))
		assert.Equal(t, MAX_RESOURCE_LOG_FILE_COUNT_LIMIT, GetResourceLogFileCountLimit())

		// Log Size
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_LOG_SIZE_LIMIT, strconv.FormatInt(MIN_RESOURCE_LOG_SIZE_LIMIT, 10)}))
		assert.Equal(t, MIN_RESOURCE_LOG_SIZE_LIMIT, GetResourceLogSizeLimit())
		assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_LOG_SIZE_LIMIT, strconv.FormatInt(MAX_RESOURCE_LOG_SIZE_LIMIT, 10)}))
		assert.Equal(t, MAX_RESOURCE_LOG_SIZE_LIMIT, GetResourceLogSizeLimit())
	})

	t.Run("out of range value tests", func(t *testing.T) {
		tests := []struct {
			key           string
			value         string
			expectedError string
		}{
			{TASK_CONCURRENCY_HARDLIMIT, "0", "config value is out of valid range"},          // < min
			{TASK_CONCURRENCY_HARDLIMIT, "3147483647", "config value is out of valid range"}, // > max
			{RESOURCE_MEM_LIMIT, "-1", "invalid size unit, should be empty or B/KB/MB"},
			{RESOURCE_MEM_LIMIT, "50TB", "invalid size unit, should be empty or B/KB/MB"},
			{RESOURCE_MEM_LIMIT, "10485760", "config value is out of valid range"},        // < min
			{RESOURCE_MEM_LIMIT, "1101004800", "config value is out of valid range"},      // > max
			{RESOURCE_CPU_LIMIT, "0.05", "config value is out of valid range"},            // < min
			{RESOURCE_CPU_LIMIT, "100.0", "config value is out of valid range"},           // > max
			{RESOURCE_LOG_FILE_COUNT_LIMIT, "0", "config value is out of valid range"},    // < min
			{RESOURCE_LOG_FILE_COUNT_LIMIT, "1000", "config value is out of valid range"}, // > max
			{RESOURCE_LOG_SIZE_LIMIT, "104857", "config value is out of valid range"},     // < min
			{RESOURCE_LOG_SIZE_LIMIT, "1101004800", "config value is out of valid range"}, // > max
			{RESOURCE_OVERLOAD_LIMIT, "2", "config value is out of valid range"},          // < min
			{RESOURCE_OVERLOAD_LIMIT, "3147483647", "config value is out of valid range"}, // > max
		}

		for _, tt := range tests {
			err := UpdateConfigRuntime(logrus.New(), []string{tt.key, tt.value})
			assert.Error(t, err)
			assert.Equal(t, tt.expectedError, err.Error())
		}
	})

	t.Run("invalid config name", func(t *testing.T) {
		err := UpdateConfigRuntime(logrus.New(), []string{"undefined_key", "4"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), errUnknownConfigName.Error())
	})
}

func TestConfigPriority_SilentNoOverride(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()

	var (
		//updator_update            bool
		task_concurrencyHardLimit int64
		resource_memLimit         int64
		resource_cpuLimit         float64
		resource_overloadLimit    int64
	)

	logger := logrus.New()

	// 初始化运行时配置
	InitConfig(logger)

	// 注册回调
	RegisterCallbackAndApply(logger, map[string]Callback{
		//UPDATOR_UPDATE:             func(logger logrus.FieldLogger, v any) { updator_update, _ = v.(bool) },
		TASK_CONCURRENCY_HARDLIMIT: func(logger logrus.FieldLogger, v any) { task_concurrencyHardLimit, _ = v.(int64) },
		RESOURCE_CPU_LIMIT: func(logger logrus.FieldLogger, v any) {
			resource_cpuLimit, _ = v.(float64)
		},
		RESOURCE_MEM_LIMIT:      func(logger logrus.FieldLogger, v any) { resource_memLimit, _ = v.(int64) },
		RESOURCE_OVERLOAD_LIMIT: func(logger logrus.FieldLogger, v any) { resource_overloadLimit, _ = v.(int64) },
	})

	t.Run("runtime_high_priority_blocks_lower_sources", func(t *testing.T) {
		// Step 1: 用 Runtime 设置 (P=4)
		err := UpdateConfigRuntime(logger, []string{RESOURCE_MEM_LIMIT, "41943040"})
		assert.Nil(t, err)
		assert.Equal(t, int64(41943040), resource_memLimit)
		assert.Equal(t, int64(41943040), GetResourceMemLimit())
		assert.Equal(t, PriorityRuntime, GetResourceMemLimitPriority())

		// Step 2: 尝试通过 API 更新 (P=1) —— 应被忽略
		err = UpdateConfigApi(logger, map[string]string{RESOURCE_MEM_LIMIT: "52428800"})
		assert.Nil(t, err) // API 不会返回错误，只是跳过
		assert.Equal(t, int64(41943040), resource_memLimit, "value should not be changed by lower priority source")
		assert.Equal(t, int64(41943040), GetResourceMemLimit())
		assert.Equal(t, PriorityRuntime, GetResourceMemLimitPriority())

		// Step 3: 再次用 Runtime 更新设置 (P=4)
		err = UpdateConfigRuntime(logger, []string{RESOURCE_MEM_LIMIT, "52428800"})
		assert.Nil(t, err)
		assert.Equal(t, int64(52428800), resource_memLimit)
		assert.Equal(t, int64(52428800), GetResourceMemLimit())
		assert.Equal(t, PriorityRuntime, GetResourceMemLimitPriority())
	})

	t.Run("config_file_cannot_override_runtime", func(t *testing.T) {
		// 初始优先级为PriorityDefault
		assert.Equal(t, PriorityDefault, GetResourceCpuLimitPriority())

		// 模拟配置文件写入（P=3）
		createFile(filepath.Join(confDir, disableUpdateFlagFilename))
		err := UpdateConfigFile(logger, []string{RESOURCE_CPU_LIMIT, "85.1", RESOURCE_MEM_LIMIT, "40MB"}, false)
		assert.Nil(t, err)
		conf := loadAllConfig(logrus.New())
		assert.InDelta(t, float64(85.1), conf[RESOURCE_CPU_LIMIT].(*floatVar).GetValue(), 0.001)
		ReloadConfig(logrus.New())
		assert.Equal(t, PriorityConfigFile, GetResourceCpuLimitPriority())
		assert.InDelta(t, float64(85.1), resource_cpuLimit, 0.001)
		assert.Equal(t, PriorityConfigFile, GetResourceMemLimitPriority())
		assert.Equal(t, int64(41943040), resource_memLimit)

		// Runtime 升级为 P=4
		UpdateConfigRuntime(logger, []string{RESOURCE_CPU_LIMIT, "50", RESOURCE_OVERLOAD_LIMIT, "10", RESOURCE_MEM_LIMIT, "50MB"})
		assert.InDelta(t, float64(50), GetResourceCpuLimit(), 0.001)
		assert.InDelta(t, float64(50), resource_cpuLimit, 0.001)
		assert.Equal(t, int64(10), resource_overloadLimit)
		assert.Equal(t, int64(52428800), resource_memLimit)

		// 再次尝试通过配置文件改回来（比如用户手动编辑并 reload）
		err = UpdateConfigFile(logger, []string{RESOURCE_CPU_LIMIT, "40"}, false)
		assert.Nil(t, err)

		assert.InDelta(t, float64(50), GetResourceCpuLimit(), 0.001, "config file should not override runtime setting")
		//再次reload
		ReloadConfig(logrus.New())
		assert.Equal(t, PriorityConfigFile, GetResourceCpuLimitPriority())
		assert.InDelta(t, float64(40), resource_cpuLimit, 0.001)
		assert.Equal(t, int64(3), resource_overloadLimit)
		assert.Equal(t, PriorityDefault, GetResourceOverloadLimitPriority())
		assert.Equal(t, int64(3), GetResourceOverloadLimit())
		assert.Equal(t, int64(41943040), resource_memLimit)
	})

	t.Run("reload_conf_default_cannot_override_api", func(t *testing.T) {
		// 初始优先级为PriorityDefault
		UpdateConfigRuntime(logger, []string{RESOURCE_CPU_LIMIT, "40"})
		assert.InDelta(t, float64(40), resource_cpuLimit, 0.001)

		err := UpdateConfigApi(logger, map[string]string{TASK_CONCURRENCY_HARDLIMIT: "400"})
		assert.Nil(t, err)
		assert.Equal(t, int64(400), task_concurrencyHardLimit)
		// 模拟配置文件写入（P=3）
		createFile(filepath.Join(confDir, disableUpdateFlagFilename))
		err = UpdateConfigFile(logger, []string{RESOURCE_CPU_LIMIT, "85.1"}, false)
		assert.Nil(t, err)
		ReloadConfig(logrus.New())
		assert.InDelta(t, float64(85.1), resource_cpuLimit, 0.001)
		assert.Equal(t, int64(400), task_concurrencyHardLimit)

		os.Remove(filepath.Join(confDir, disableUpdateFlagFilename))
		ReloadConfig(logrus.New())
		assert.Equal(t, int64(400), task_concurrencyHardLimit)
		assert.InDelta(t, float64(85.1), resource_cpuLimit, 0.001)
	})

	t.Run("api_can_only_update_if_no_higher_priority_set", func(t *testing.T) {
		// 假设初始 priority 是 PriorityDefault (0)，那么 API(P=1) 可以更新
		err := UpdateConfigApi(logger, map[string]string{TASK_CONCURRENCY_HARDLIMIT: "200"})
		assert.Nil(t, err)
		assert.Equal(t, int64(200), task_concurrencyHardLimit)
		assert.True(t, GetTaskConcurrencyHardlimit() == 200)
		assert.True(t, GetTaskConcurrencyHardlimitPriority() <= PriorityApi) // 至少 <=1

		// 假设初始 priority 是 PriorityDefault (0)，那么 API(P=1) 可以更新
		err = UpdateConfigApi(logger, map[string]string{TASK_CONCURRENCY_HARDLIMIT: "400"})
		assert.Nil(t, err)
		assert.Equal(t, int64(400), task_concurrencyHardLimit)
		assert.True(t, GetTaskConcurrencyHardlimit() == 400)
		assert.True(t, GetTaskConcurrencyHardlimitPriority() <= PriorityApi) // 至少 <=1

		// 现在用 Runtime 提权到 P=4
		UpdateConfigRuntime(logger, []string{TASK_CONCURRENCY_HARDLIMIT, "100"})
		assert.Equal(t, int64(100), task_concurrencyHardLimit)

		// 再尝试用 API 改回
		err = UpdateConfigApi(logger, map[string]string{TASK_CONCURRENCY_HARDLIMIT: "999"})
		assert.Nil(t, err)
		assert.Equal(t, int64(100), task_concurrencyHardLimit, "API should not override higher-priority runtime config")
	})
}

func TestDumpConfig(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()
	InitConfig(logrus.New())

	dumpConfigFile(logrus.New(), _runtimeConfig, false)
	content, err := os.ReadFile(filepath.Join(confDir, configurationFile))
	assert.Nil(t, err)
	contentStr := string(content)
	// fmt.Println(contentStr)
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", UPDATOR_BOOTSTRAP_UPDATE, VALUE_ENABLED))
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", UPDATOR_UPDATE, VALUE_ENABLED))
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", TASK_CONCURRENCY_HARDLIMIT, DEFAULT_TASK_CONCURRENCY_HARDLIMIT))
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", TASK_KEEP_SCRIPT_FILE, VALUE_DISABLED))
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", RESOURCE_CPU_LIMIT, DEFAULT_RESOURCE_CPU_LIMIT))
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", RESOURCE_MEM_LIMIT, DEFAULT_RESOURCE_MEM_LIMIT))
	assert.Contains(t, contentStr, fmt.Sprintf("%s=%v", RESOURCE_OVERLOAD_LIMIT, DEFAULT_RESOURCE_OVERLOAD_LIMIT))
}

func TestLoadConfig(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()
	InitConfig(logrus.New())

	conf := loadConfigFile(logrus.New(), true, false)
	assert.Zero(t, len(conf))
	conf = loadConfigFile(logrus.New(), false, true)
	assert.Zero(t, len(conf))
	conf = loadConfigFile(logrus.New(), true, true)
	assert.Zero(t, len(conf))

	var confStr string
	confStr += "undefined=abc\n"
	confStr += fmt.Sprintf("%s=%v\n", TASK_CONCURRENCY_HARDLIMIT, 101)
	confStr += fmt.Sprintf("%s=%v\n", RESOURCE_CPU_LIMIT, 21.1)

	var confCrossVerStr string
	confCrossVerStr += "undefined=abc\n"
	confCrossVerStr += fmt.Sprintf("%s=%v\n", TASK_CONCURRENCY_HARDLIMIT, 102)
	confCrossVerStr += fmt.Sprintf("%s=%v\n", RESOURCE_MEM_LIMIT, 104857600)

	err := os.WriteFile(filepath.Join(confDir, configurationFile), []byte(confStr), os.ModePerm)
	assert.Nil(t, err)
	err = os.WriteFile(filepath.Join(confCrossVerDir, configurationFile), []byte(confCrossVerStr), os.ModePerm)
	assert.Nil(t, err)

	conf = loadConfigFile(logrus.New(), true, false)
	assert.Equal(t, 2, len(conf))
	assert.Equal(t, int64(101), conf[TASK_CONCURRENCY_HARDLIMIT].(*intVar).GetValue())
	assert.Equal(t, float64(21.1), conf[RESOURCE_CPU_LIMIT].(*floatVar).GetValue())
	conf = loadConfigFile(logrus.New(), false, true)
	assert.Equal(t, 2, len(conf))
	assert.Equal(t, int64(102), conf[TASK_CONCURRENCY_HARDLIMIT].(*intVar).GetValue())
	assert.Equal(t, int64(104857600), conf[RESOURCE_MEM_LIMIT].(*intVarWithSizeUnit).GetValue())
	conf = loadConfigFile(logrus.New(), true, true)
	assert.Equal(t, 3, len(conf))
	assert.Equal(t, int64(101), conf[TASK_CONCURRENCY_HARDLIMIT].(*intVar).GetValue())
	assert.Equal(t, float64(21.1), conf[RESOURCE_CPU_LIMIT].(*floatVar).GetValue())
	assert.Equal(t, int64(104857600), conf[RESOURCE_MEM_LIMIT].(*intVarWithSizeUnit).GetValue())
}

func Test_Compatibility(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()
	InitConfig(logrus.New())

	// Identify flag files
	createFile(filepath.Join(confDir, disableBootstrapUpdateFlagFilename))
	conf := loadConfigFile(logrus.New(), true, false)
	assert.Equal(t, false, conf[UPDATOR_BOOTSTRAP_UPDATE].(*boolVar).GetValue())
	assert.Nil(t, conf[UPDATOR_UPDATE])
	createFile(filepath.Join(confDir, disableUpdateFlagFilename))
	conf = loadConfigFile(logrus.New(), true, false)
	assert.Equal(t, false, conf[UPDATOR_BOOTSTRAP_UPDATE].(*boolVar).GetValue())
	assert.Equal(t, false, conf[UPDATOR_UPDATE].(*boolVar).GetValue())
	conf = loadConfigFile(logrus.New(), false, true)
	assert.Nil(t, conf[UPDATOR_BOOTSTRAP_UPDATE])
	assert.Nil(t, conf[UPDATOR_UPDATE])

	os.Remove(filepath.Join(confDir, disableBootstrapUpdateFlagFilename))
	os.Remove(filepath.Join(confDir, disableUpdateFlagFilename))

	createFile(filepath.Join(confCrossVerDir, disableBootstrapUpdateFlagFilename))
	conf = loadConfigFile(logrus.New(), false, true)
	assert.Equal(t, false, conf[UPDATOR_BOOTSTRAP_UPDATE].(*boolVar).GetValue())
	assert.Nil(t, conf[UPDATOR_UPDATE])
	createFile(filepath.Join(confCrossVerDir, disableUpdateFlagFilename))
	conf = loadConfigFile(logrus.New(), false, true)
	assert.Equal(t, false, conf[UPDATOR_BOOTSTRAP_UPDATE].(*boolVar).GetValue())
	assert.Equal(t, false, conf[UPDATOR_UPDATE].(*boolVar).GetValue())
	conf = loadConfigFile(logrus.New(), true, false)
	assert.Nil(t, conf[UPDATOR_BOOTSTRAP_UPDATE])
	assert.Nil(t, conf[UPDATOR_UPDATE])

	os.Remove(filepath.Join(confCrossVerDir, disableBootstrapUpdateFlagFilename))
	os.Remove(filepath.Join(confCrossVerDir, disableUpdateFlagFilename))

	createFile(filepath.Join(confDir, disableUpdateFlagFilename))
	UpdateConfigFile(logrus.New(), []string{UPDATOR_UPDATE, VALUE_ENABLED}, false)
	conf = loadAllConfig(logrus.New())
	assert.Equal(t, true, conf[UPDATOR_UPDATE].(*boolVar).GetValue())
}

func Test_LoadConfigFromMetaserver(t *testing.T) {
	guard := gomonkey.ApplyFunc(httpbase.Get, func(url string, requestOptions ...httpbase.RequestOption) (string, error) {
		return `assistDaemon.active=disabled
apiserver.tryPreset=disabled`, nil
	})
	defer guard.Reset()
	InitConfig(logrus.New())

	assert.False(t, GetAssistDaemonActive())
	assert.False(t, GetApiserverTryPreset())
}

func mockConfDir(t *testing.T) (string, string, *gomonkey.Patches, *gomonkey.Patches) {
	current, err := os.Executable()
	assert.Nil(t, err)
	current = filepath.Dir(current)

	confDir := filepath.Join(current, "config")
	confCrossVerDir := filepath.Join(current, "configCrossVer")
	os.RemoveAll(confDir)
	os.RemoveAll(confCrossVerDir)
	os.Mkdir(confDir, os.ModePerm)
	os.Mkdir(confCrossVerDir, os.ModePerm)

	guard_1 := gomonkey.ApplyFunc(pathutil.GetConfigPath, func() (string, error) {
		return confDir, nil
	})
	guard_2 := gomonkey.ApplyFunc(pathutil.GetCrossVersionConfigPath, func() (string, error) {
		return confCrossVerDir, nil
	})

	return confDir, confCrossVerDir, guard_1, guard_2
}

func createFile(filePath string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func TestParseConfigFromFeature(t *testing.T) {
	tests := []struct {
		content     string
		expectItems map[string]string
		expectErr   string
	}{
		{
			content:     `{"features":{"task":[{}}`,
			expectErr:   "invalid json response",
			expectItems: map[string]string{},
		},
		{
			content:     `{"code":200,"instanceId":"i-123","config":null}`,
			expectItems: map[string]string{},
		},
		{
			content:     `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[]}}}`,
			expectItems: map[string]string{},
		},
		{
			content:     `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"websocket"}],"update":null,"resource":null}}}`,
			expectItems: map[string]string{"task.protocol": "websocket"},
		},
		{
			content:     `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"gshell"}],"update":null,"resource":null}}}`,
			expectItems: map[string]string{"task.protocol": "gshell"},
		},
		{
			content:     `{"code":200,"instanceId":"i-123","config":{"feature":{"resource":[{"field":"cpuLimit","value":30}]}}}`,
			expectItems: map[string]string{"resource.cpuLimit": "30"},
		},
		{
			content:     `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"gshell"}],"resource":[{"field":"cpuLimit1","value":30}]}}}`, //contian invalid field
			expectItems: map[string]string{"task.protocol": "gshell"},
		},
	}

	for _, ele := range tests {
		items, err := ParseConfigFromFeature(logrus.New(), ele.content)
		if ele.expectErr == "" {
			assert.Equal(t, err, nil)
		} else {
			assert.Equal(t, err.Error(), ele.expectErr)
		}
		assert.Equal(t, ele.expectItems, items)
	}
}

func TestUpdateConfOnConfigReceived(t *testing.T) {
	t.Run("ShouldCallUpdateConfigApiWithNonProtocolKeys", func(t *testing.T) {
		content := `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"gshell"}],"resource":[{"field":"cpuLimit","value":30},{"field":"memLimit","value":"50MB"}]}}}`
		featureItems := map[string]string{
			TASK_PROTOCOL:      "gshell",
			RESOURCE_CPU_LIMIT: "30",
			RESOURCE_MEM_LIMIT: "50MB",
		}
		patches := gomonkey.NewPatches()
		defer patches.Reset()
		patches.ApplyFunc(UpdateConfigApi,
			func(logger logrus.FieldLogger, toUpdate map[string]string) error {
				assert.Equal(t, len(toUpdate), len(featureItems))
				for key, _ := range toUpdate {
					assert.Equal(t, toUpdate[key], featureItems[key])
				}
				return nil
			},
		)

		UpdateConfOnConfigReceived(logrus.New(), content)
	})

	t.Run("ShouldNotCallUpdateConfigApiWhenNoValidKeys", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		updateCalled := false
		patches.ApplyFunc(
			UpdateConfigApi,
			func(logger logrus.FieldLogger, toUpdate map[string]string) error {
				updateCalled = true
				return nil
			},
		)
		tests := []string{
			`{"code":200,"instanceId":"i-123","config":{"feature":{"task":[]}}}`,
			`{"features":{"task":[{}}`,
		}
		for _, test := range tests {
			UpdateConfOnConfigReceived(logrus.New(), test)
		}
		assert.False(t, updateCalled, "UpdateConfigApi should not be called when no valid keys")
	})

	t.Run("ShouldHandleFailureAndLogError", func(t *testing.T) {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(UpdateConfigApi, func(logger logrus.FieldLogger, toUpdate map[string]string) error {
			return errors.New("update failed")
		})

		content := `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"gshell"}],"resource":[{"field":"cpuLimit","value":30},{"field":"memLimit","value":"50MB"}]}}}`
		require.NotPanics(t, func() {
			UpdateConfOnConfigReceived(logrus.New(), content)
		})
	})
}

func TestOneTimeConfigAndCallback(t *testing.T) {
	confDir, confCrossVerDir, g1, g2 := mockConfDir(t)
	defer func() {
		g1.Reset()
		g2.Reset()
		os.RemoveAll(confDir)
		os.RemoveAll(confCrossVerDir)
	}()

	var (
		protocal string
		isCalled bool
	)

	logger := logrus.New()

	// 初始化运行时配置
	InitConfig(logger)

	// 注册回调
	RegisterCallbackAndApply(logger, map[string]Callback{
		TASK_PROTOCOL: func(logger logrus.FieldLogger, v any) {
			protocal, _ = v.(string)
			isCalled = true
		},
	})
	assert.Equal(t, protocal, "")
	assert.Equal(t, isCalled, false)
	t.Run("ShouldCallWebsocket", func(t *testing.T) {
		content := `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"websocket"}],"update":null,"resource":null}}}`
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		UpdateConfOnConfigReceived(logrus.New(), content)
		assert.Equal(t, protocal, "websocket")
		assert.Equal(t, isCalled, true)
	})

	t.Run("ShouldCallGshell", func(t *testing.T) {
		content := `{"code":200,"instanceId":"i-123","config":{"feature":{"task":[{"field":"protocol","value":"gshell"}],"update":null,"resource":null}}}`
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		protocal = ""
		isCalled = false
		UpdateConfOnConfigReceived(logrus.New(), content)
		assert.Equal(t, protocal, "gshell")
		assert.Equal(t, isCalled, true)
	})

	t.Run("ShouldNotCall", func(t *testing.T) {
		content := `{"code":200,"instanceId":"i-123","config":{"feature":{"resource":[{"field":"cpuLimit","value":30},{"field":"memLimit","value":"50MB"}]}}}`
		protocal = ""
		isCalled = false
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		UpdateConfOnConfigReceived(logrus.New(), content)
		assert.Equal(t, protocal, "")
		assert.Equal(t, isCalled, false)
	})
}
