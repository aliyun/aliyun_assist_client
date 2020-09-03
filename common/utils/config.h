#ifndef PROJECT_CONFIG_H_
#define PROJECT_CONFIG_H_

#include <string.h>
#include <stdio.h>
#include <iostream>

using std::string;
class AssistConfig {
 public:
  static std::string GetConfigValue(std::string key, std::string val);

 private:
  static void LoadConfigDatas();
  static void ParseConfigInfos(std::string data);
};

#endif //PROJECT_CONFIG_H_


