#include "config.h"

#include <mutex>
#include <map>

#include "utils/MutexLocker.h"
#include "Log.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "json11/json11.h"

std::map<std::string, std::string> config_datas;
bool data_loaded = false;

void AssistConfig::LoadConfigDatas() {
  data_loaded = true;
  AssistPath path("");
  std::string config_path = path.GetConfigPath();
  std::string config_file_path = config_path + FileUtils::separator() + "config.json";

  if(!FileUtils::fileExists(config_file_path.c_str())) {
    return;
  }

  std::string content;
  FileUtils::readFile(config_file_path, content);
  ParseConfigInfos(content);
}

void AssistConfig::ParseConfigInfos(std::string data) {
  Log::Info("Load config datas");
  try {
    string errinfo;
    auto json_datas = json11::Json::parse(data, errinfo);
    if (errinfo != "") {
	    Log::Error("invalid json format");
	    return;
    }
    for(auto v : json_datas.object_items()) {
      config_datas.insert(std::pair<std::string, std::string>(v.first, v.second.string_value()));
    }
  }
  catch (...) {
    Log::Error("Parse config string failed, response:%s", data.c_str());
  }
}

std::string AssistConfig::GetConfigValue(std::string key, std::string val)  {
  static std::mutex  mutex;
  MutexLocker(&mutex) {
    if (!data_loaded) {
      LoadConfigDatas();
    }

    if(!config_datas.size()) {
      return val;
    }

    if(config_datas.find(key) == config_datas.end()) {
      return val;
    }

    return config_datas[key];
  }
  return val;
}

