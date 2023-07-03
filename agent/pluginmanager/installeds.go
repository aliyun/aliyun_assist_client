package pluginmanager

import (
	"os"

	"github.com/aliyun/aliyun_assist_client/thirdparty/sirupsen/logrus"

	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
	"github.com/aliyun/aliyun_assist_client/common/fuzzyjson"
)

// InstalledPlugins 和installed_plugins文件内容一致，用于解析json
type InstalledPlugins struct {
	PluginList []PluginInfo `json:"pluginList"`
}

func LoadInstalledPlugins() (*InstalledPlugins, error) {
	jsonPath, err := getInstalledPluginsJSONPath()
	if err != nil {
		return nil, err
	}
	installedPlugins := InstalledPlugins{}

	if !util.CheckFileIsExist(jsonPath) {
		return &installedPlugins, nil
	}

	content, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	if err := fuzzyjson.Unmarshal(string(content), &installedPlugins); err != nil {
		log.GetLogger().WithFields(logrus.Fields{
			"content": content,
		}).WithError(err).Errorln("Failed to unmarshal JSON database file of install plugins")
	}

	return &installedPlugins, nil
}

func (ip *InstalledPlugins) FindAll() ([]int, []PluginInfo) {
	keys := make([]int, len(ip.PluginList))
	for k := 0; k < len(ip.PluginList); k++ {
		keys[k] = k
	}

	return keys, ip.PluginList
}

func (ip *InstalledPlugins) FindManyByName(name string) ([]int, []PluginInfo) {
	keys := []int{}
	found := []PluginInfo{}
	for k, v := range ip.PluginList {
		if v.Name != name {
			continue
		}

		keys = append(keys, k)
		found = append(found, v)
	}

	return keys, found
}

func (ip *InstalledPlugins) FindOneByName(name string) (int, *PluginInfo) {
	for i := 0; i < len(ip.PluginList); i++ {
		if ip.PluginList[i].Name == name {
			return i, &ip.PluginList[i]
		}
	}

	return -1, nil
}

func (ip *InstalledPlugins) FindOneNotRemovedByName(name string) (int, *PluginInfo) {
	for i := 0; i < len(ip.PluginList); i++ {
		if ip.PluginList[i].IsRemoved {
			continue
		}

		if ip.PluginList[i].Name == name {
			return i, &ip.PluginList[i]
		}
	}

	return -1, nil
}

func (ip *InstalledPlugins) FindOneNotRemovedByNameAndOptionalVersion(name string, version string) (int, *PluginInfo) {
	for i := 0; i < len(ip.PluginList); i++ {
		if ip.PluginList[i].IsRemoved {
			continue
		}

		if ip.PluginList[i].Name == name {
			if version == "" || ip.PluginList[i].Version == version {
				return i, &ip.PluginList[i]
			}
		}
	}

	return -1, nil
}

func (ip *InstalledPlugins) Insert(value *PluginInfo) int {
	ip.PluginList = append(ip.PluginList, *value)
	return len(ip.PluginList) - 1
}

// Update method simply stores new value to specified position in JSON array.
// Would PANIC if key is out of range.
func (ip *InstalledPlugins) Update(key int, value *PluginInfo) {
	ip.PluginList[key] = *value
}

func (ip *InstalledPlugins) DeleteByKey(key int) {
	ip.PluginList = append(ip.PluginList[:key], ip.PluginList[key+1:]...)
}

func (ip *InstalledPlugins) Save() error {
	jsonPath, err := getInstalledPluginsJSONPath()
	if err != nil {
		return err
	}

	content, err := fuzzyjson.Marshal(ip)
	if err != nil {
		return err
	}

	return os.WriteFile(jsonPath, []byte(content), os.FileMode(0o0640))
}
