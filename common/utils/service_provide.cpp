// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./service_provide.h"

#include <string>

#include "utils/host_finder.h"
#include "utils/Log.h"

std::string ServiceProvide::GetUpdateService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/update/update_check";
  return url;
}

std::string ServiceProvide::GetConnectDetectService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/connection_detect";
  return url;
}

//New version 1.0
std::string ServiceProvide::GetInvalidTaskService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/invalid";
  return url;
}

std::string ServiceProvide::GetFetchTaskListService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/list";
  return url;
}

std::string ServiceProvide::GetRunningOutputService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/running";
  return url;
}

std::string ServiceProvide::GetFinishOutputService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/finish";
  return url;
}

std::string ServiceProvide::GetStoppedOutputService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/stopped";
  return url;
}

std::string ServiceProvide::GetTimeoutOutputService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/timeout";
  return url;
}

std::string ServiceProvide::GetErrorOutputService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/error";
  return url;
}

std::string ServiceProvide::GetPingService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/heart-beat";
  return url;
}

std::string ServiceProvide::GetGshellCheckService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/gshell";
  return url;
}

std::string ServiceProvide::GetPluginListService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/plugin/list";
  return url;
}
