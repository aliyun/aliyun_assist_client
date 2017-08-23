// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./fetch_task.h"

#include <vector>
#include <string>

#include "jsoncpp/json.h"
#include "utils/http_request.h"
#include "utils/service_provide.h"
#include "utils/CheckNet.h"
#include "utils/Encode.h"
#include "utils/Log.h"

namespace {
void parse_task_info(std::string response,
    std::vector<task_engine::TaskInfo>& task_info) {
  Json::Value jsonRoot;
  Json::Value jsonValue;
  Json::Reader reader;
  if (!reader.parse(response, jsonRoot)) {
    Log::Error("invalid json format");
    return;
  }

  if (jsonRoot.isArray()) {
    for (unsigned int i = 0; i < jsonRoot.size(); i++) {
      task_engine::TaskInfo info;
      if(jsonRoot[i]["taskInstanceID"].isString())
        info.instance_id = jsonRoot[i]["taskInstanceID"].asString();
      jsonValue = jsonRoot[i]["taskItem"];
      if (jsonValue.empty())
        return;
      info.command_id = jsonValue["type"].asString();
      info.task_id = jsonValue["taskID"].asString();
      std::string content = jsonValue["commandContent"].asString();
      Encoder encode;
      if (!content.empty()) {
        info.content = reinterpret_cast<char *>(encode.B64Decode(
            content.c_str(), content.size()));
      }
      info.working_dir = jsonValue["workingDirectory"].asString();
      info.cronat = jsonValue["cron"].asString();
      info.time_out = jsonValue["timeOut"].asString();
      task_info.push_back(info);
    }
  }
}
}  // namespace

namespace task_engine {
TaskFetch::TaskFetch() {
}

void TaskFetch::FetchTasks(std::vector<TaskInfo>& task_info) {
  std::string response;
  if(HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetFetchTaskService();
  HttpRequest::http_request_post(url, "", response);
  parse_task_info(response, task_info);
  Log::Info("response:%s", response.c_str());
  Log::Info("Fetch_Tasks:Fetch %d Tasks", task_info.size());
}

#if defined(TEST_MODE)
void TaskFetch::TestFetchTasks(std::string res, std::vector<TaskInfo>& task_info) {
  parse_task_info(res, task_info);
}
#endif

void TaskFetch::FetchCancledTasks(std::vector<TaskInfo>& task_info) {
  std::string response;
  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetFetchCanceledTaskService();
  HttpRequest::http_request_post(url, "", response);
  parse_task_info(response, task_info);
  Log::Info("Fetch_Cancled_Tasks:Fetch %d Tasks", task_info.size());
}

void TaskFetch::FetchPeriodTasks(std::vector<TaskInfo>& task_info) {
  std::string response;
  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetFetchPeriondTaskService();
  HttpRequest::http_request_post(url, "", response);
  parse_task_info(response, task_info);
  Log::Info("Fetch_Period_Tasks:Fetch %d Tasks", task_info.size());
}

}  // namespace task_engine
