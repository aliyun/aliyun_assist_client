// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./fetch_task.h"

#include <vector>
#include <string>

#include "utils/http_request.h"
#include "utils/service_provide.h"
#include "utils/host_finder.h"
#include "utils/encoder.h"
#include "utils/Log.h"
#include "utils/AssistPath.h"
#include "json11/json11.h"

namespace {

void ParseStopTaskInfo(json11::Json json,
  std::vector<task_engine::StopTaskInfo>& stop_task_info)
{
  try {
    for (auto &it : json.array_items()) {
      task_engine::StopTaskInfo info;

      auto taskjson = it["task"];
      if (taskjson.is_null()) {
        Log::Error("stop task json is null");
        continue;
      }

      info.json_data = taskjson.dump();
      info.instance_id = taskjson["instanceId"].string_value();
      info.command_type = taskjson["type"].string_value();
      info.task_id = taskjson["taskID"].string_value();
      info.command_id = taskjson["commandId"].string_value();
      info.command_name = taskjson["commandName"].string_value();
      std::string content = taskjson["commandContent"].string_value();
      Encoder encode;
      if (!content.empty()) {
        info.content = reinterpret_cast<char *>(encode.B64Decode(
          content.c_str(), content.size()));
      }
      info.working_dir = taskjson["workingDirectory"].string_value();
      info.args = taskjson["args"].string_value();
      info.cronat = taskjson["cron"].string_value();
      info.time_out = taskjson["timeOut"].string_value();
      if (info.time_out.empty()) {
        info.time_out = "3600";
      }
      stop_task_info.push_back(info);
    }
  }
  catch (...) {
    Log::Error("Stop task list json is invalid");
  }
}

void ParseRunTaskInfo(json11::Json json,
  std::vector<task_engine::RunTaskInfo>& run_task_info)
{
  try {
    for (auto &it : json.array_items()) {
      task_engine::RunTaskInfo info;

      auto taskjson = it["task"];
      if (taskjson.is_null()) {
        Log::Error("run task json is null");
        continue;
      }

      info.json_data = taskjson.dump();
      info.instance_id = taskjson["instanceId"].string_value();
      info.command_type = taskjson["type"].string_value();
      info.task_id = taskjson["taskID"].string_value();
      info.command_id = taskjson["commandId"].string_value();
      info.command_name = taskjson["commandName"].string_value();
      std::string content = taskjson["commandContent"].string_value();
      Encoder encode;
      if (!content.empty()) {
        info.content = reinterpret_cast<char *>(encode.B64Decode(
          content.c_str(), content.size()));
      }
      info.working_dir = taskjson["workingDirectory"].string_value();
      info.args = taskjson["args"].string_value();
      info.cronat = taskjson["cron"].string_value();
      info.time_out = taskjson["timeOut"].string_value();
      if (info.time_out.empty()) {
        info.time_out = "3600";
      }

      auto outputjson = it["output"];
      if (outputjson.is_null()) {
        Log::Error("run task output json is null");
        continue;
      }

      info.output_info.interval = outputjson["interval"].int_value();
      if (info.output_info.interval == 0) {
        info.output_info.interval = 3000;
      }
      info.output_info.log_quota = outputjson["logQuota"].int_value();
      if (info.output_info.log_quota == 0) {
        info.output_info.log_quota = 12288;
      }
      info.output_info.skip_empty = outputjson["skipEmpty"].bool_value();
      info.output_info.send_start = outputjson["sendStart"].bool_value();

      run_task_info.push_back(info);
    }
  }
  catch (...) {
    Log::Error("Run task list json is invalid");
  }
}

void ParseTaskInfo(std::string response,
  std::vector<task_engine::StopTaskInfo>& stop_task_info,
  std::vector<task_engine::RunTaskInfo>& run_task_info) {

  try {
    string errinfo;
    auto json = json11::Json::parse(response, errinfo);
    if (errinfo != "") {
      Log::Error("invalid json format");
      return;
    }

    ParseStopTaskInfo(json["stop"], stop_task_info);
    ParseRunTaskInfo(json["run"], run_task_info);
  }
  catch (...) {
    Log::Error("Task list json is invalid");
  }
}
}  // namespace

namespace task_engine {
TaskFetch::TaskFetch() {
}

/*#if defined(TEST_MODE)
void TaskFetch::TestFetchTasks(std::string res, std::vector<TaskInfo>& task_info) {
  parse_task_info(res, task_info);
}
#endif*/

void TaskFetch::FetchTaskList(std::vector<task_engine::StopTaskInfo>& stop_task_info,
    std::vector<task_engine::RunTaskInfo>& run_task_info,
    std::string reason) {

  if (HostFinder::getServerHost().empty()) {
    return;
  }
  std::string response;
  std::string url = ServiceProvide::GetFetchTaskListService();
  Log::Info("fetch-task request {\"method\": \"GET\", \"url\" : \"%s\", \"parameters\" : {\"reason\": \"%s\"} }", url.c_str(), reason.c_str());

  url = url + "?reason=" + reason;
  bool ret = HttpRequest::https_request_post(url, "", response);
  if (!ret) {
    Log::Error("fetch-task response %s", response.c_str());
  }

  for (int i = 0; i < 10 && !ret; i++) {
    int second = int(pow(2, i));
    std::this_thread::sleep_for(std::chrono::seconds(second));
    ret = HttpRequest::https_request_post(url, "", response);
    if (!ret) {
      Log::Error("fetch-task response %s", response.c_str());
    }
  }

  if (!ret)
    return;

  Log::Info("fetch-task response %s", response.c_str());
  ParseTaskInfo(response, stop_task_info, run_task_info);
}

}  // namespace task_engine
