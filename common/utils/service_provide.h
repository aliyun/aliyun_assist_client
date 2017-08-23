// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef COMMON_UTILS_SERVICE_PROVIDE_H_
#define COMMON_UTILS_SERVICE_PROVIDE_H_

#include <string>

class ServiceProvide {
 public:
  static std::string GetUpdateService();
  static std::string GetFetchTaskService();
  static std::string GetFetchCanceledTaskService();
  static std::string GetFetchPeriondTaskService();
  static std::string GetReportTaskStatusService();
  static std::string GetReportTaskOutputService();
};
#endif  // COMMON_UTILS_SERVICE_PROVIDE_H_
