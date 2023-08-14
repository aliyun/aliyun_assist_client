package acspluginmanager

import (
	"github.com/aliyun/aliyun_assist_client/agent/pluginmanager"
)

func getAllInstalledPlugins() ([]pluginmanager.PluginInfo, error) {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}
	defer installedPlugins.Close()

	_, plugins, err := installedPlugins.FindAll()
	return plugins, err
}

func getInstalledPluginsByName(name string) ([]pluginmanager.PluginInfo, error) {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}
	defer installedPlugins.Close()

	_, plugins, err := installedPlugins.FindManyByName(name)
	return plugins, err
}

func getInstalledPluginByName(name string) (int, *pluginmanager.PluginInfo, error) {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return -1, nil, err
	}
	defer installedPlugins.Close()

	return installedPlugins.FindOneWithPredicate(func(plugin *pluginmanager.PluginInfo) bool {
		return plugin.Name == name
	})
}

func getInstalledPluginNotRemovedByName(name string) (int, *pluginmanager.PluginInfo, error) {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return -1, nil, err
	}
	defer installedPlugins.Close()

	return installedPlugins.FindOneWithPredicate(func(plugin *pluginmanager.PluginInfo) bool {
		return !plugin.IsRemoved && plugin.Name == name
	})
}

func getLocalPluginInfo(packageName, pluginVersion string) (*pluginmanager.PluginInfo, error) {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}
	defer installedPlugins.Close()

	_, pluginInfo, err := installedPlugins.FindOneWithPredicate(func(plugin *pluginmanager.PluginInfo) bool {
		return !plugin.IsRemoved && plugin.Name == packageName &&
			(pluginVersion == "" || plugin.Version == pluginVersion)
	})
	return pluginInfo, err
}

func insertNewInstalledPlugin(plugin *pluginmanager.PluginInfo) (int, error) {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return -1, err
	}
	defer installedPlugins.Close()

	return installedPlugins.Insert(plugin)
}

func updateInstalledPlugin(idx int, plugin *pluginmanager.PluginInfo) error {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return err
	}
	defer installedPlugins.Close()

	return installedPlugins.Update(idx, plugin)
}

func deleteInstalledPluginByIndex(idx int) error {
	installedPlugins, err := pluginmanager.LoadInstalledPlugins()
	if err != nil {
		return err
	}
	defer installedPlugins.Close()

	return installedPlugins.DeleteByKey(idx)
}
