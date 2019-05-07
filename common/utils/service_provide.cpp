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

std::string ServiceProvide::GetFetchTaskService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/fetch_task";
  return url;
}

std::string ServiceProvide::GetFetchCanceledTaskService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/fetch_canceled_task";
  return url;
}

std::string ServiceProvide::GetFetchPeriondTaskService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/fetch_active_period_task";
  return url;
}

std::string ServiceProvide::GetReportTaskStatusService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/update_status";
  return url;
}

std::string ServiceProvide::GetReportTaskOutputService() {
  std::string url = "https://" + HostFinder::getServerHost();
  url += "/luban/api/v1/task/upload_output";
  return url;
}
