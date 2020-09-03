// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef COMMON_UTILS_SERVICE_PROVIDE_H_
#define COMMON_UTILS_SERVICE_PROVIDE_H_

#include <string>

class ServiceProvide {
 public:
  static std::string GetUpdateService();

  // New version 1.0
  static std::string GetInvalidTaskService();
  static std::string GetConnectDetectService();
  static std::string GetFetchTaskListService();
  static std::string GetRunningOutputService();
  static std::string GetFinishOutputService();
  static std::string GetStoppedOutputService();
  static std::string GetTimeoutOutputService();
  static std::string GetErrorOutputService();
  static std::string GetPingService();
  static std::string GetGshellCheckService();
  static std::string GetPluginListService();
};
#endif  // COMMON_UTILS_SERVICE_PROVIDE_H_
