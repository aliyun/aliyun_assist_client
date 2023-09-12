package statemanager

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/pathutil"
)

var (
	_stateConfigCacheLock   sync.Mutex
	stateConfigCache *ListInstanceStateConfigurationsResult
)


func ConfigCacheFilePath() (path string, err error) {
	cacheDir, err := pathutil.GetCachePath()
	if err != nil {
		log.GetLogger().WithError(err).Errorln("get path failed")
		return
	}
	path = filepath.Join(cacheDir, "state_configs.json")
	return
}

// LoadConfigCache loads state configurations from local cache files 
func LoadConfigCache() (r *ListInstanceStateConfigurationsResult, err error) {
	_stateConfigCacheLock.Lock()
	defer _stateConfigCacheLock.Unlock()
	if stateConfigCache != nil {
		log.GetLogger().Debug("got state configuration cache in memory")
		return stateConfigCache, nil
	}
	path, err := ConfigCacheFilePath()
	if err != nil {
		return 
	}
	if !util.CheckFileIsExist(path) {
		return
	}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.GetLogger().WithError(err).Errorln("read cache file error")
		return
	}
	var result ListInstanceStateConfigurationsResult
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}
	stateConfigCache = &result
	log.GetLogger().Debug("loaded state configuration from file")
	return stateConfigCache, nil
}

// WriteConfigCache saves state configurations to local cache file
func WriteConfigCache(config *ListInstanceStateConfigurationsResult) error {
	_stateConfigCacheLock.Lock()
	defer _stateConfigCacheLock.Unlock()
	stateConfigCache = config
	path, err := ConfigCacheFilePath()
	if err != nil {
		return nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		log.GetLogger().WithError(err).Errorln("marshal config to json error")
		return err
	}
	log.GetLogger().Debugf("saving state configuration to %s", path)
	err = ioutil.WriteFile(path, data, os.ModePerm)
	return err
}

func TemplateCachePath(name string, version string) (p string, err error) {
	cacheDir, err := pathutil.GetCachePath()
	if err != nil {
		log.GetLogger().WithError(err).Errorln("get template cache path failed")
		return
	}
	templateDir := filepath.Join(cacheDir, "template")
	pathutil.MakeSurePath(templateDir)
	p = filepath.Join(cacheDir, "template", name + "_" + version + ".json")
	return
}

func LoadTemplateCache(name string, version string) (content []byte, err error) {
	path, err := TemplateCachePath(name, version)
	if err != nil {
		return 
	}
	if !util.CheckFileIsExist(path) {
		return
	}
	return ioutil.ReadFile(path)
}

func WriteTemplateCache(name string, version string, data []byte) (err error) {
	path, err := TemplateCachePath(name, version)
	if err != nil {
		return 
	}
	log.GetLogger().Debugf("saving template to %s", path)
	return ioutil.WriteFile(path, data, os.ModePerm)
}

func LastCheckpoint() string {
	lastResult, err := LoadConfigCache()
	if err != nil {
		return ""
	} else {
		return lastResult.Checkpoint
	}
}
