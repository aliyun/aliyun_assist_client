// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./task.h"

#include <string>

#include "utils/CheckNet.h"
#include "jsoncpp/json.h"
#include "utils/http_request.h"
#include "utils/Encode.h"
#include "utils/Log.h"
#include "utils/service_provide.h"

#if !defined(_WIN32)
#include<sys/types.h>
#include<signal.h>
#endif

namespace task_engine {
Task::Task(TaskInfo info) :sub_process_(task_info_.working_dir,
    atoi(task_info_.time_out.c_str())) {
  task_info_ = info;
  err_code_ = 0;
  is_timeout = false;
  is_period_ = !task_info_.cronat.empty();
  Log::Info("taskid:%s command_id:%s content:%s params:%s", \
      task_info_.task_id.c_str(), task_info_.command_id.c_str(),
      task_info_.content.c_str(), task_info_.params.c_str());
}

Task::Task() : sub_process_("", 3600){
  is_timeout = false;
}

void Task::Run() {
}

void Task::Cancel() {
  Log::Info("cancel the task:%s", task_info_.task_id.c_str());
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

void Task::CheckTimeout() {
#if defined(_WIN32)
  HANDLE hprocess = sub_process_.get_id();
  DWORD ExitCode = 0;
  ::GetExitCodeProcess(hprocess, &ExitCode);
  if(ExitCode == STILL_ACTIVE) {
    Log::Info("process is timeout");
    is_timeout = true;
    ::TerminateProcess(hprocess, 0);
  } else {
    Log::Info("process is not timeout");
  }
#else
  pid_t pid= sub_process_.get_id();
  if(kill(pid, 0) != 0) {
    Log::Info("process is not timeout");
  } else {
    Log::Info("process is timeout");
    is_timeout = true;
    kill(pid, SIGKILL);
   }
#endif
}

void Task::ReportOutput() {
  if(is_timeout) {
    ReportTimeout();
    return;
  }
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

void Task::ReportTimeout() {
  Log::Info("Report timeout");

  status_ = "failed";

  std::string response;
  std::string input;
  Json::Value jsonRoot;
  Json::Value jsonOutput;

  jsonOutput["taskInstanceOutput"] = "";
  jsonRoot["taskID"] = task_info_.task_id;
  jsonOutput["errNo"] = (int)err_code_;
  jsonRoot["taskStatus"] = status_;
  jsonRoot["taskOutput"] = "";
  input = jsonRoot.toStyledString();

  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetReportTaskOutputService();
  HttpRequest::http_request_post(url, input, response);
}

}  // namespace task_engine
