// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./task.h"

#include <string>

#include "utils/CheckNet.h"
#include "jsoncpp/json.h"
#include "utils/http_request.h"
#include "utils/Encode.h"
#include "utils/Log.h"
#include "utils/service_provide.h"

namespace task_engine {
Task::Task(TaskInfo info) :sub_process_(task_info_.working_dir,
    atoi(task_info_.time_out.c_str())) {
  task_info_ = info;
  err_code_ = 0;
#if defined(_WIN32)
  process_id = nullptr;
#else
  process_id = 0;
#endif
  is_period_ = !task_info_.cronat.empty();
  Log::Info("taskid:%s command_id:%s content:%s params:%s", \
      task_info_.task_id.c_str(), task_info_.command_id.c_str(),
      task_info_.content.c_str(), task_info_.params.c_str());
}

void Task::Run() {
}

void Task::Cancel() {
  Log::Info("cancel the task:%s", task_info_.task_id.c_str());
#if defined(_WIN32)
  if (sub_process_.get_id())
    ::TerminateProcess(sub_process_.get_id(), -1);
#else
#endif
  ReportStatus("stopped");
}

void Task::ReportStatus(std::string status, std::string instance_id) {
  status_ = status;
  std::string response;
  std::string input;
  Json::Value jsonRoot;
  if (!instance_id.empty()) {
    jsonRoot["taskInstanceID"] = task_info_.instance_id;
  }
  jsonRoot["taskStatus"] = status_;
  jsonRoot["taskID"] = task_info_.task_id;

  input = jsonRoot.toStyledString();
  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetReportTaskStatusService();
  HttpRequest::http_request_post(url, input, response);
  Log::Info("ReportStatus input:%s", input.c_str());
  Log::Info("ReportStatus status:%s", status.c_str());
}

void Task::ReportOutput() {
  status_ = "finished";

  std::string response;
  std::string input;
  Json::Value jsonRoot;
  Json::Value jsonOutput;
  Encoder encoder;
  char* pencodedata = encoder.B64Encode(
      (const unsigned char *)task_output_.c_str(), task_output_.size());
  jsonOutput["taskInstanceOutput"] = pencodedata;
  free(pencodedata);
  jsonRoot["taskID"] = task_info_.task_id;
  jsonOutput["errNo"] = (int)err_code_;
  jsonRoot["taskStatus"] = status_;
  jsonRoot["taskOutput"] = jsonOutput;
  input = jsonRoot.toStyledString();

  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetReportTaskOutputService();
  HttpRequest::http_request_post(url, input, response);

  Log::Info("ReportOutput input:%s", input.c_str());
  Log::Info("Report taskid:%s task_output:%s error_code:%d %s:response",
      task_info_.task_id.c_str(), task_output_.c_str(),
      err_code_, response.c_str());
}
}  // namespace task_engine
