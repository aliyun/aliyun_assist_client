package flagging

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"

	"github.com/aliyun/aliyun_assist_client/common/httpbase"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

		apiserver_try_presets bool
		assistDaemon_active   bool
	)
	InitConfig(logrus.New())

	RegisterCallbackAndApply(logrus.New(), map[string]Callback{
		UPDATOR_BOOTSTRAP_UPDATE: func(logger logrus.FieldLogger, v any) { updator_bootstrapUpdate, _ = v.(bool) },
		UPDATOR_UPDATE:           func(logger logrus.FieldLogger, v any) { updator_update, _ = v.(bool) },

		TASK_CONCURRENCY_HARDLIMIT: func(logger logrus.FieldLogger, v any) { task_concurrencyHardLimit, _ = v.(int64) },
		TASK_KEEP_SCRIPT_FILE:      func(logger logrus.FieldLogger, v any) { task_keepScriptFile, _ = v.(bool) },

		RESOURCE_CPU_LIMIT:      func(logger logrus.FieldLogger, v any) { resource_cpuLimit, _ = v.(float64) },
		RESOURCE_MEM_LIMIT:      func(logger logrus.FieldLogger, v any) { resource_memLimit, _ = v.(int64) },
		RESOURCE_OVERLOAD_LIMIT: func(logger logrus.FieldLogger, v any) { resource_overloadLimit, _ = v.(int64) },

		APISERVER_TRY_PRESETS: func(logger logrus.FieldLogger, v any) { apiserver_try_presets, _ = v.(bool) },
		ASSIST_DAEMON_ACTIVE: func(logger logrus.FieldLogger, v any) { assistDaemon_active, _ = v.(bool) },
	})
	assert.Equal(t, DEFAULT_UPDATOR_BOOTSTRAP_UPDATE, updator_bootstrapUpdate)
	assert.Equal(t, DEFAULT_UPDATOR_UPDATE, updator_update)
	assert.Equal(t, int64(DEFAULT_TASK_CONCURRENCY_HARDLIMIT), task_concurrencyHardLimit)
	assert.Equal(t, DEFAULT_TASK_KEEP_SCRIPT_FILE, task_keepScriptFile)
	assert.Equal(t, DEFAULT_RESOURCE_CPU_LIMIT, resource_cpuLimit)
	assert.Equal(t, int64(DEFAULT_RESOURCE_MEM_LIMIT), resource_memLimit)
	assert.Equal(t, int64(DEFAULT_RESOURCE_OVERLOAD_LIMIT), resource_overloadLimit)
	assert.Equal(t, DEFAULT_APISERVER_TRY_PRESETS, apiserver_try_presets)
	assert.Equal(t, DEFAULT_ASSIST_DAEMON_ACTIVE, assistDaemon_active)

	assert.Equal(t, DEFAULT_UPDATOR_BOOTSTRAP_UPDATE, GetUpdatorBootstrapUpdate())
	assert.Equal(t, DEFAULT_UPDATOR_UPDATE, GetUpdatorUpdate())
	assert.Equal(t, int64(DEFAULT_TASK_CONCURRENCY_HARDLIMIT), GetTaskConcurrencyHardlimit())
	assert.Equal(t, DEFAULT_TASK_KEEP_SCRIPT_FILE, GetTaskKeepScriptFile())
	assert.Equal(t, DEFAULT_RESOURCE_CPU_LIMIT, GetResourceCpuLimit())
	assert.Equal(t, int64(DEFAULT_RESOURCE_MEM_LIMIT), GetResourceMemLimit())
	assert.Equal(t, int64(DEFAULT_RESOURCE_OVERLOAD_LIMIT), GetResourceOverloadLimit())
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
	)

	InitConfig(logrus.New())
	RegisterCallbackAndApply(logrus.New(), map[string]Callback{
		UPDATOR_BOOTSTRAP_UPDATE: func(logger logrus.FieldLogger, v any) { updator_bootstrapUpdate, _ = v.(bool) },
		UPDATOR_UPDATE:           func(logger logrus.FieldLogger, v any) { updator_update, _ = v.(bool) },

		TASK_CONCURRENCY_HARDLIMIT: func(logger logrus.FieldLogger, v any) { task_concurrencyHardLimit, _ = v.(int64) },
		TASK_KEEP_SCRIPT_FILE:      func(logger logrus.FieldLogger, v any) { task_keepScriptFile, _ = v.(bool) },

		RESOURCE_CPU_LIMIT:      func(logger logrus.FieldLogger, v any) { resource_cpuLimit, _ = v.(float64) },
		RESOURCE_MEM_LIMIT:      func(logger logrus.FieldLogger, v any) { resource_memLimit, _ = v.(int64) },
		RESOURCE_OVERLOAD_LIMIT: func(logger logrus.FieldLogger, v any) { resource_overloadLimit, _ = v.(int64) },
	})

	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{UPDATOR_BOOTSTRAP_UPDATE, VALUE_DISABLED}))
	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{UPDATOR_UPDATE, VALUE_DISABLED}))
	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{TASK_CONCURRENCY_HARDLIMIT, "501"}))
	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{TASK_KEEP_SCRIPT_FILE, VALUE_ENABLED}))
	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_CPU_LIMIT, "21.1"}))
	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_MEM_LIMIT, "51"}))
	assert.Nil(t, UpdateConfigRuntime(logrus.New(), []string{RESOURCE_OVERLOAD_LIMIT, "4"}))
	assert.ErrorContains(t, UpdateConfigRuntime(logrus.New(), []string{"undifined", "4"}), errUnknownConfigName.Error())

	assert.Equal(t, false, updator_bootstrapUpdate)
	assert.Equal(t, false, updator_update)
	assert.Equal(t, int64(501), task_concurrencyHardLimit)
	assert.Equal(t, true, task_keepScriptFile)
	assert.Equal(t, float64(21.1), resource_cpuLimit)
	assert.Equal(t, int64(51), resource_memLimit)
	assert.Equal(t, int64(4), resource_overloadLimit)
	assert.Equal(t, false, GetUpdatorBootstrapUpdate())
	assert.Equal(t, false, GetUpdatorUpdate())
	assert.Equal(t, int64(501), GetTaskConcurrencyHardlimit())
	assert.Equal(t, true, GetTaskKeepScriptFile())
	assert.Equal(t, float64(21.1), GetResourceCpuLimit())
	assert.Equal(t, int64(51), GetResourceMemLimit())
	assert.Equal(t, int64(4), GetResourceOverloadLimit())
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
	confCrossVerStr += fmt.Sprintf("%s=%v\n", RESOURCE_MEM_LIMIT, 51)

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
	assert.Equal(t, int64(51), conf[RESOURCE_MEM_LIMIT].(*intVar).GetValue())
	conf = loadConfigFile(logrus.New(), true, true)
	assert.Equal(t, 3, len(conf))
	assert.Equal(t, int64(101), conf[TASK_CONCURRENCY_HARDLIMIT].(*intVar).GetValue())
	assert.Equal(t, float64(21.1), conf[RESOURCE_CPU_LIMIT].(*floatVar).GetValue())
	assert.Equal(t, int64(51), conf[RESOURCE_MEM_LIMIT].(*intVar).GetValue())
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
