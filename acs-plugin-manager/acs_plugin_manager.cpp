// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./acs_plugin_manager.h"
#include <string>
#include <vector>
#include <algorithm>
#ifdef _WIN32
#include <windows.h>
#else
#include <unistd.h>
#include <sys/wait.h>
#endif

#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/process.h"
#include "utils/OsVersion.h"
#include "utils/FileVersion.h"
#include "utils/host_finder.h"
#include "utils/http_request.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"
#include "utils/VersionComparator.h"
#include "utils/service_provide.h"

#include "zip/zip.h"
#include "md5/md5.h"
#include "bprinter/table_printer.h"
using bprinter::TablePrinter;

namespace acs {
PluginManager::PluginManager(bool verbose):verbose_(verbose) {
  
}

PluginManager::~PluginManager() {

}

json11::Json PluginInfo::to_json() const {
  json11::Json json_value = json11::Json::object {
    { "pluginId", pluginId },
    { "name", name },
    { "arch", arch},
    { "osType", osType },
    { "version", version },
    { "publisher", publisher},
    { "url", url },
    { "md5", md5 },
    { "runPath", runPath},
    { "timeout", timeout },
    { "isPreInstalled", isPreInstalled},
};
  return json_value;
}


void PluginManager::LoadPackages() {
  AssistPath path("");
  std::string plugin_path = path.GetPluginPath();

  std::string plugin_lists_path = plugin_path + FileUtils::separator() + "installed_plugins";

  if(!FileUtils::fileExists(plugin_lists_path.c_str())) {
    return;
  }

  std::string content;
  FileUtils::readFile(plugin_lists_path, content);
  installed_packages_ = ParsePluginInfos(content);
}

void PluginManager::UpdatePackages(PluginInfo plugin_info) {
  bool found = false;
  for(auto &value:installed_packages_) {
    if(value.name == plugin_info.name){
      found = true;
////////////////////////////////////////////////////////////////////
      value.publisher = plugin_info.publisher;
      value.url = plugin_info.url;
      value.md5 = plugin_info.md5;
      value.timeout = plugin_info.timeout;
      value.runPath = plugin_info.runPath;
      value.version = plugin_info.version;
      value.isPreInstalled = plugin_info.isPreInstalled;
      value.osType = plugin_info.osType;
      value.arch = plugin_info.arch;
    }
  }
  if(!found) {
    installed_packages_.push_back(plugin_info);
  }

}

void PluginManager::SavePackages() {
  AssistPath path("");
  std::string plugin_path = path.GetPluginPath();
  std::string plugin_lists_path = plugin_path + FileUtils::separator() + "installed_plugins";
 // std::string content = json11::Json(installed_packages_).dump();
  json11::Json plugin_json = json11::Json::object {
    { "pluginList", installed_packages_ },
  };
  std::string json_str = plugin_json.dump();
  FileUtils::writeFile(plugin_lists_path, json_str);
}

void print_plugin_info(std::vector<PluginInfo> package_infos) {
    TablePrinter tp(&std::cout);
    tp.AddColumn("Name", 30);
    tp.AddColumn("version", 10);
    tp.AddColumn("publisher", 10);
    tp.AddColumn("os", 10);

    tp.PrintHeader();

    for (size_t i = 0; i < package_infos.size(); ++i) {
      tp << package_infos[i].name.c_str() << package_infos[i].version.c_str() << package_infos[i].publisher.c_str() << package_infos[i].osType.c_str();
    }
    tp.PrintFooter();

}

int PluginManager::ListLocal() {
  LoadPackages();
  if(installed_packages_.size() == 0) {
    printf("there is no plugin in local");
    return 0;
  }
  print_plugin_info(installed_packages_);
  return 0;
}

int PluginManager::List(std::string pluginName) {
  Log::Info("list plugins");

  vector<PluginInfo> package_infos = GetPackageInfo(pluginName);

  if (package_infos.empty()) {
    printf("There is no plugin in the plugin repository.\n");
    return 0;
  } else {
    print_plugin_info(package_infos);
  }
  return 0;
}

std::vector<PluginInfo> PluginManager::ParsePluginInfos(
    std::string response) {
  Log::Info("ParsePluginInfos");
  std::vector<PluginInfo> plugin_infos;
  try {
	  string errinfo;
	  auto json = json11::Json::parse(response, errinfo);
	  if (errinfo != "") {
		  Log::Error("invalid json format");
		  return plugin_infos;
	  }

    auto plugin_data = json["pluginList"];
	  for ( auto &it : plugin_data.array_items() ) {
        PluginInfo plugin_info;
        plugin_info.pluginId = it["pluginId"].string_value();
        plugin_info.name = it["name"].string_value();
        plugin_info.publisher = it["publisher"].string_value();
        plugin_info.url = it["url"].string_value();
        plugin_info.md5 = it["md5"].string_value();
        plugin_info.timeout = it["timeout"].string_value();
        plugin_info.runPath = it["runPath"].string_value();
        plugin_info.version = it["version"].string_value();
        plugin_info.isPreInstalled = it["isPreInstalled"].string_value();
        plugin_info.osType = it["osType"].string_value();
        plugin_info.arch = it["arch"].string_value();
        std::transform(plugin_info.md5.begin(), plugin_info.md5.end(),
           plugin_info.md5.begin(), ::tolower);
        plugin_infos.push_back(plugin_info);
    }
  }
  catch (...) {
    Log::Error("ParseResponseString failed, response:%s",
        response.c_str());
  }

  return plugin_infos;
}

int PluginManager::installPlugin(const std::string& package_name,  std::string params, std::string separator) {
  PluginInfo localInfo;
  PluginInfo RemoteInfo;
  if(separator.empty()) {
    separator = ",";
  }
  std::replace(params.begin(), params.end(), separator[0], ' ');
  bool local_found = getLocalPluginInfo(package_name, localInfo);
  bool remote_found = getOnlinePluginInfo(package_name, RemoteInfo);

  bool using_local = false;

  if(local_found) {
    if(remote_found) {
      if(localInfo.version.compare(RemoteInfo.version) == 0 ){
        using_local = true;
      }

    } else {
      using_local = true;
    }

  } else {
    if(!remote_found) {
      printf("could not found the package");
      return 1;
    }  
  }

  int ret = 0;

  if(using_local) {
    if (verbose_)
      printf("using local plugins");
    ret =  InstallActionLocal(localInfo, params);
  } else {
    if (verbose_)
      printf("using remote plugins");
    ret = InstallAction(RemoteInfo, params);
    if(ret == 0) {
      UpdatePackages(RemoteInfo);
      SavePackages();
    }
  }

  return ret;
}

bool PluginManager::getLocalPluginInfo(const std::string& package_name, PluginInfo& plugin_info) {
  LoadPackages();
  for(auto value:installed_packages_) {
    if(value.name == package_name) {
      plugin_info = value;
      return true;
    }
  }
  return false;
}

bool PluginManager::getOnlinePluginInfo(const std::string& package_name, PluginInfo& plugin_info) {
  std::vector<PluginInfo> plugins = GetPackageInfo(package_name);
  for(auto value:plugins) {
    if(value.name == package_name) {
      plugin_info = value;
      return true;
    }
  }
  return false;
}


int PluginManager::InstallActionLocal(const PluginInfo& plugin_info, const std::string& params) {
  Log::Info("Enter InstallAction Local");
  AssistPath path("");
  std::string plugin_path = path.GetPluginPath();
  std::string unzip_path = plugin_path + FileUtils::separator();
  unzip_path += plugin_info.name;
  unzip_path += FileUtils::separator();
  unzip_path += plugin_info.version;
  std::string cmd_path = unzip_path + FileUtils::separator() + plugin_info.runPath;

  if(!FileUtils::fileExists(cmd_path.c_str())) {
    printf("error, could not find the run exec path of the plugin.\n");
    return 1;
  }

#if !defined _WIN32
  std::string cmd = "chmod 744 " + cmd_path;
  system(cmd.c_str());
#endif

  int timeout = 60;
  if(!plugin_info.timeout.empty()) {
    timeout = atoi(plugin_info.timeout.c_str());
  }

  std::string commond;
  commond = cmd_path + " " + params;

  if(verbose_) {
    printf("run command:%s",commond.c_str());
  }

  int exit_code = 0;
  auto callback = [this]( const char* buf, size_t len ) {
    printf("%s", buf);
  };
  Process(commond).syncRun(timeout, callback, callback, &exit_code);

  return exit_code;
}

int PluginManager::InstallAction(const PluginInfo& plugin_info, const std::string& params) {
  Log::Info("Enter InstallAction");
  AssistPath path("");
  std::string plugin_path = path.GetPluginPath();

  std::string file_path = plugin_path;
  std::string file_name = plugin_info.url.substr(
      plugin_info.url.find_last_of('/') + 1);
  file_path += FileUtils::separator();
  file_path.append(file_name);
  Log::Info("Call Download, %s", plugin_info.url.c_str());
 
  if(verbose_)
    printf("Downloading...\n");
  bool download_ret = Download(plugin_info.url, file_path);
  if (!download_ret) {
    printf("Download this package failed, %s.\n", plugin_info.url.c_str());
    return 1;
  }

  if(verbose_)
    printf("Check MD5...\n");
  if (!CheckMd5(file_path, plugin_info.md5)) {
    if(verbose_)
      printf("Check file md5 failed.\n");
    return 1;
  }

  std::string unzip_path = plugin_path + FileUtils::separator();
  unzip_path += plugin_info.name;
  unzip_path += FileUtils::separator();
  unzip_path += plugin_info.version;
  path.MakeSurePath(unzip_path);

  if(verbose_)
    printf("Unzip to %s...\n", unzip_path.c_str());
  bool unzip_ret = UnZip(file_path, unzip_path);
  if (!unzip_ret) {
    printf("Unzip this package failed, please try again later.\n");
    return 1;
  }

  FileUtils::removeFile(file_path.c_str());

  std::string cmd_path = unzip_path + FileUtils::separator() + plugin_info.runPath;

  if(!FileUtils::fileExists(cmd_path.c_str())) {
    printf("error, could not find the run exec path of the plugin.\n");
    return 1;
  }

#if !defined _WIN32
  std::string cmd = "chmod 744 " + cmd_path;
  system(cmd.c_str());
#endif

  int timeout = 60;
  if(!plugin_info.timeout.empty()) {
    timeout = atoi(plugin_info.timeout.c_str());
  }

  std::string commond;
  commond = cmd_path + " " + params;

  if(verbose_) {
    printf("run command:%s",commond.c_str());
  }

  int exit_code = 0;
  auto callback = [this]( const char* buf, size_t len ) {
    printf("%s",buf);
  };
  Process(commond).syncRun(timeout, callback, callback, &exit_code);

  return exit_code;
}

int PluginManager::verifyPlugin(const std::string& url, std::string params, std::string separator) {
  Log::Info("Enter InstallAction");
  AssistPath path("");
  std::string plugin_path = path.GetPluginPath();

  std::string file_path = plugin_path;
  std::string file_name = url.substr(url.find_last_of('/') + 1);
  file_path += FileUtils::separator();
  file_path.append(file_name);
  Log::Info("Call Download, %s", url.c_str());
 
  if(verbose_)
    printf("Downloading...\n");
  bool download_ret = Download(url, file_path);
  if (!download_ret) {
    printf("Download this package failed, %s.\n", url.c_str());
    return 1;
  }


  std::string unzip_path = plugin_path + FileUtils::separator();
  unzip_path += "verify_plugin_test";

  path.MakeSurePath(unzip_path);

  if(verbose_)
    printf("Unzip to %s...\n", unzip_path.c_str());
  bool unzip_ret = UnZip(file_path, unzip_path);
  if (!unzip_ret) {
    printf("Unzip this package failed, please try again later.\n");
    return 1;
  }

  FileUtils::removeFile(file_path.c_str());

  std::string config_path = unzip_path + FileUtils::separator() + "config.json";

  if(!FileUtils::fileExists(config_path.c_str())) {
    printf("can not find the config.json"); 
    return 1;
  }

  std::string content;
  FileUtils::readFile(config_path, content);

  std::string runPath;

  try {
	  string errinfo;
	  auto json = json11::Json::parse(content, errinfo);
	  if (errinfo != "") {
		  Log::Error("invalid json format");
		  return 1;
	  }

    runPath = json["runPath"].string_value();;

  }
  catch (...) {
    Log::Error("ParseResponseString failed, response:%s",
        content.c_str());
  }

    // get the cmd path
  std::string cmd_path = unzip_path + FileUtils::separator() + runPath;

  if(!FileUtils::fileExists(cmd_path.c_str())) {
    printf("error, could not find the run exec path of the plugin.\n");
    return 1;
  }

#if !defined _WIN32
  std::string cmd = "chmod 744 " + cmd_path;
  system(cmd.c_str());
#endif

  int timeout = 60;

  std::string commond;
  if(separator.empty()) {
    separator = ",";
  }

  std::replace(params.begin(), params.end(), separator[0], ' ');
  commond = cmd_path + " " + params;

  if(verbose_) {
    printf("run command:%s",commond.c_str());
  }

  int exit_code = 0;
  auto callback = [this]( const char* buf, size_t len ) {
    printf("%s",buf);
  };
  Process(commond).syncRun(timeout, callback, callback, &exit_code);

  return exit_code;
}


vector<PluginInfo> PluginManager::GetPackageInfo(std::string pluginName) {
  Log::Info("GetPackageInfo");
  std::string response;
  std::vector<PluginInfo> package_infos;

  json11::Json  json = json11::Json::object {
  #ifdef _WIN32
  {"osType","windows"},
  #else
  {"osType", "linux"},
  #endif
  { "pluginName", pluginName }
  };

  std::string post_value = json.dump();

  bool ret = HttpRequest::https_request_post(ServiceProvide::GetPluginListService(), post_value, response);

  if(verbose_)
    printf(response.c_str());
  Log::Info("response:%s", response.c_str());
  if (ret) {
    package_infos = ParsePluginInfos(response);
  } else {
    printf("http request failed,%s",response.c_str());
  }

  return package_infos;
}

bool PluginManager::Download(const std::string& url,
    const std::string& path) {
  Log::Info("Enter Download, %s", url.c_str());
  bool ret = HttpRequest::download_file(url, path);
  if (ret) {
    return true;
  } else {
    if (verbose_)
      printf("Download failed, url: %s", url.c_str());
    Log::Error("Download failed, url: %s", url.c_str());
    return false;
  }
}

bool PluginManager::CheckMd5(const std::string& path,
    const std::string& md5_string) {

	std::string origal = md5_string;
	transform(origal.begin(), origal.end(), origal.begin(), (int(*)(int))tolower);

	std::string file_md5 = md5file(path.c_str());
	transform(file_md5.begin(), file_md5.end(), file_md5.begin(), (int(*)(int))tolower);

	if (origal.compare(file_md5) == 0) {
		return true;
	}
	else {
		return false;
	}
}

bool PluginManager::UnZip(const std::string& file_name,
    const std::string& dir) {
  int ret = zip_extract(file_name.c_str(), dir.c_str(), nullptr, nullptr);
  if (ret == 0) {
    return true;
  } else {
    Log::Error("UnZip failed, file name: %s", file_name.c_str());
    if (verbose_)
      printf("UnZip failed, %s", file_name.c_str());
    return false;
  }
}

}  // namespace alyun_assist_installer
