// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#ifndef PLUGIN_PACKAGEMANAGER_PACKAGEMANAGER_H_
#define PLUGIN_PACKAGEMANAGER_PACKAGEMANAGER_H_

#include <vector>
#include <string>
#include "json11/json11.h"

namespace acs {
  
struct PluginInfo{
  std::string pluginId;
  std::string name;
  std::string arch;
  std::string osType;
  std::string version;
  std::string publisher;
  std::string url;
  std::string md5;
  std::string runPath;
  std::string timeout;
  std::string isPreInstalled;
  json11::Json to_json() const;
};
  
class PluginManager {
 public:
  PluginManager(bool verbose);
  virtual ~PluginManager();
  int List(std::string pluginName);
  int ListLocal();
  int installPlugin(const std::string& package_name, std::string params, std::string separator);
  int verifyPlugin(const std::string& url, std::string params, std::string separator);
 private:
  std::vector<PluginInfo> ParsePluginInfos(std::string response);
  std::vector<PluginInfo>  GetPackageInfo(std::string pluginName);
  int InstallAction(const PluginInfo& plugin_info, const std::string& params);
  int InstallActionLocal(const PluginInfo& plugin_info, const std::string& params);
  bool Download(const std::string& url, const std::string& path);
  bool CheckMd5(const std::string& path, const std::string& md5_string);
  bool UnZip(const std::string& file_name, const std::string& dir);
  void LoadPackages();
  void SavePackages();
  void UpdatePackages(PluginInfo plugin_info);
  bool getLocalPluginInfo(const std::string& package_name, PluginInfo& plugin_info);
  bool getOnlinePluginInfo(const std::string& package_name, PluginInfo& plugin_info);

  bool verbose_;
  std::vector<PluginInfo> installed_packages_;

};

}  // namespace acs

#endif  // PLUGIN_PACKAGEMANAGER_PACKAGEMANAGER_H_
