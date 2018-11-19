// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./task.h"

#include <string>

#include "utils/CheckNet.h"
#include "jsoncpp/json.h"
#include "utils/http_request.h"
#include "utils/Encode.h"
#include "utils/Log.h"
#include "utils/service_provide.h"
#include "utils/singleton.h"
#include "plugin/timeout_listener.h"

#if !defined(_WIN32)
#include<sys/types.h>
#include<signal.h>
#include <sys/wait.h>
#endif

namespace task_engine {

void upload_retry_callback(void * context) {
  Log::Error("upload ouput failed, add retry");
  Task* task = reinterpret_cast<Task*>(context);
  if(!task) {
    Log::Error("task is nullptr");
    return;
  }
  if(task->getRetryNum() > 0) {
    task->setRetryNum(task->getRetryNum() - 1);
    task->ReportOutput();
  }
 
}

Task::Task(TaskInfo info) :sub_process_(info.working_dir,
    atoi(info.time_out.c_str())) {
  task_info_ = info;
  err_code_ = 0;
  retry_num_ = 3;
  is_timeout = false;
  is_reported = false;
  is_period_ = !task_info_.cronat.empty();
  Log::Info("taskid:%s command_id:%s content:%s params:%s", \
      task_info_.task_id.c_str(), task_info_.command_id.c_str(),
      task_info_.content.c_str(), task_info_.params.c_str());
}

Task::Task() : sub_process_("", 3600){
  is_timeout = false;
  is_reported = false;
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
  HttpRequest::https_request_post(url, input, response);
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
    ReportTimeout();
   }
#endif
}

void Task::ReportOutput() {
  if (HostChooser::m_HostSelect.empty()) {
    return;
  }

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

  std::string url = ServiceProvide::GetReportTaskOutputService();
  bool ret = HttpRequest::https_request_post(url, input, response);

  Log::Info("ReportOutput input:%s", input.c_str());
  Log::Info("Report taskid:%s task_output:%s error_code:%d %s:response",
      task_info_.task_id.c_str(), task_output_.c_str(),
      err_code_, response.c_str());
  if(ret == false && retry_num_ > 0) {
      Singleton<TimeoutListener>::I().CreateTimer(
          &upload_retry_callback,
          reinterpret_cast<void*>(this), 5);
  }
}

void Task::ReportTimeout() {
  Log::Info("Report timeout");

  if (HostChooser::m_HostSelect.empty()) {
    return;
  }

  if(is_reported == true) {
    return;
  }
  is_reported = true;

  status_ = "failed";

  std::string response;
  std::string input;
  Json::Value jsonRoot;
  Json::Value jsonOutput;

  Encoder encoder;
  char* pencodedata = encoder.B64Encode(
      (const unsigned char *)task_output_.c_str(), task_output_.size());
  jsonOutput["taskInstanceOutput"] = pencodedata;
  jsonOutput["errNo"] = 0;
  jsonRoot["taskID"] = task_info_.task_id;
  jsonOutput["errNo"] = (int)err_code_;
  jsonRoot["taskStatus"] = status_;
  jsonRoot["taskOutput"] = jsonOutput;
  input = jsonRoot.toStyledString();

  std::string url = ServiceProvide::GetReportTaskOutputService();
  HttpRequest::https_request_post(url, input, response);

  Log::Info("ReportOutput input:%s", input.c_str());
  Log::Info("Report taskid:%s task_output:%s error_code:%d %s:response",
      task_info_.task_id.c_str(), task_output_.c_str(),
      err_code_, response.c_str());
}

}  // namespace task_engine
