// Copyright (c) 2017-2018 Alibaba Group Holding Limited.


#include <string>
#include <math.h>
#include <stdio.h>
#include <stdlib.h>

#include "base_task.h"
#include "utils/host_finder.h"
#include "utils/http_request.h"
#include "utils/encoder.h"
#include "utils/Log.h"
#include "utils/service_provide.h"
#include "utils/singleton.h"
#include "utils/TimeTool.h"
#include "utils/MutexLocker.h"
#include "json11/json11.h"
#include "timer_manager.h"
#include "utils/SystemInfo.h"


#if !defined(_WIN32)
#include<sys/types.h>
#include<signal.h>
#include <sys/wait.h>
#include <unistd.h>
#else
#include <windows.h>
#include "utils\CStringUtil.h"
#endif



namespace task_engine {

BaseTask::BaseTask(RunTaskInfo info):task_info(info),
    timer(nullptr),
    canceled(false),
    output_timer(nullptr),
    received(0),
    accepted(0),
    current(0),
    dropped(0),
    cmd(""),
    output(""),
    running_output(""),
    exit_code(-1) {
}

void BaseTask::DoWork(std::string cmd, std::string dir, int timeout) {
  output_timer = Singleton<TimerManager>::I().createTimer([this]() {
    SendRunningOutput();
  }, task_info.output_info.interval / 1000);
	auto callback = [this]( const char* buf, size_t len ) {
		if (!buf || 0 == len)
		{
			return;
		}
#ifdef _WIN32
		LCID lcid = SystemInfo::GetWindowsDefaultLang();
		if (lcid != 0x409) {//非英文环境才转换
			std::string tmp = buf;
			this->output += CStringUtil::AsciiToUtf8(tmp);
		}
		else {
			//英文环境不用转换
			this->output += buf;
		}
#else
		this->output += buf;
#endif
	  
    {
      MutexLocker(&m_mutex) {
        this->running_output += buf;
      }
    }
	};

  this->cmd = cmd;
  start_time = TimeTool::GetAccurateTime();
  Log::Info("start-task %s, %s", task_info.task_id.c_str(), task_info.json_data.c_str());
  SendTaskStart();
	Process::RunResult result = Process(cmd, dir).syncRun(timeout, callback, callback, &exit_code);
  Log::Info("task-result %s, %d", task_info.task_id.c_str(), result);
  end_time = TimeTool::GetAccurateTime();
  Log::Info("task-finished: %s, stop runing output", task_info.task_id.c_str());
  DeleteOutputTimer();
	if ( result == Process::timeout ) {
    SendTimeoutOutput();
	}
	else if( result == Process::fail ) {
    SendErrorOutput();
	}
	else {
    if (canceled == false)
      SendFinishOutput();
	}
};

void BaseTask::Cancel() {
  Log::Info("cancel the task:%s", task_info.task_id.c_str());
  canceled = true;
  end_time = TimeTool::GetAccurateTime();
  SendStoppedOutput();
}

void BaseTask::SendInvalidTask(std::string param, std::string value) {
  if (HostFinder::getServerHost().empty()) {
    Log::Error("Get server host failed");
    return;
  }

  std::string url = ServiceProvide::GetInvalidTaskService();
  Log::Info("invalid-task %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"param\": \"%s\", \"value\" : %s}}",
      task_info.task_id.c_str(), url.c_str(),
      param.c_str(), value.c_str());

  url = url + "?" + "taskId=" + task_info.task_id +
      "&param=" + param + "&value=" + value;
  std::string response;
  bool ret = HttpRequest::https_request_post_text(url, "", response);
  if (!ret) {
    Log::Error("task-output %s response:%s",
        task_info.task_id.c_str(), response.c_str());
    return;
  }

  Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
}

void BaseTask::SendTaskStart() {
  if (task_info.output_info.send_start == false)
    return;

  if (HostFinder::getServerHost().empty()) {
    Log::Error("Get server host failed");
    return;
  }

  std::string url = ServiceProvide::GetRunningOutputService();
  Log::Info("send-output %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"taskId\": \"%s\", \"start\" : %lld}}",
task_info.task_id.c_str(), url.c_str(),
task_info.task_id.c_str(), start_time);

  char time[25];
  sprintf(time, "%lld", start_time);
  url = url + "?" + "taskId=" + task_info.task_id + "&start=" + time;
  std::string response;
  bool ret = HttpRequest::https_request_post_text(url, "", response);
  if (!ret) {
    Log::Error("task-output %s response:%s",
      task_info.task_id.c_str(), response.c_str());
    return;
  }

  Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
}

void BaseTask::SendRunningOutput() {
  if (HostFinder::getServerHost().empty()) {
    Log::Error("Get server host failed");
    return;
  }

  MutexLocker(&m_mutex) {
    if (task_info.output_info.skip_empty && running_output.empty()) {
      Log::Info("The runnint output is empty, skipped.");
      return;
    }

    if (accepted < received) {
      Log::Info("The running output reach quota: %s, stop uploading output, running_output: %s", task_info.task_id.c_str(), running_output.c_str());
      return;
    }

    std::string url = ServiceProvide::GetRunningOutputService();
    Log::Info("send-output %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"taskId\": \"%s\", \"start\" : %lld}}, content:\n%s",
task_info.task_id.c_str(), url.c_str(),
task_info.task_id.c_str(), start_time, running_output.c_str());

    char time[25];
    sprintf(time, "%lld", start_time);
    url = url + "?" + "taskId=" + task_info.task_id + "&start=" + time;
    std::string response;
    bool ret = HttpRequest::https_request_post_text(url, running_output, response);
    if (!ret) {
      Log::Error("task-output %s response:%s",
        task_info.task_id.c_str(), response.c_str());
      return;
    }

    Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
    running_output = "";

    try {
      string errinfo;
      auto json = json11::Json::parse(response, errinfo);
      if (errinfo != "") {
        Log::Error("Invalid json format:%s", errinfo.c_str());
        return;
      }

      received = json["received"].int_value();
      accepted = json["accepted"].int_value();
      current = json["current"].int_value();
    }
    catch (...) {
      Log::Error("Response is invalid: %s", response.c_str());
    }
    
  }
}

void BaseTask::SendFinishOutput() {
  if (HostFinder::getServerHost().empty()) {
    Log::Error("Get server host failed");
    return;
  }

  MutexLocker(&m_mutex) {
    int available_quota = task_info.output_info.log_quota - current;
    if (output.size() - current > available_quota)
      dropped = output.size() - current - available_quota;
    else
      dropped = 0;

    std::string url = ServiceProvide::GetFinishOutputService();
    Log::Info("finish-task %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"taskId\": \"%s\", \"start\" : %lld, \"end\" : %lld, \
\"exitCode\" : %d, \"dropped\" : %lld}}, content:\n%s",
task_info.task_id.c_str(), url.c_str(),
task_info.task_id.c_str(), start_time, end_time,
exit_code, dropped, output.c_str());

    if (current + dropped > 0) {
      output.erase(0, current + dropped);
    }

    char param[512];
    sprintf(param, "?taskId=%s&start=%lld&end=%lld&exitCode=%d&dropped=%d",
      task_info.task_id.c_str(), start_time, end_time, exit_code, dropped);
    url += param;
    std::string response;
    bool ret = HttpRequest::https_request_post_text(url, output, response);
    if (!ret) {
      Log::Error("task-output %s %s",
        task_info.task_id.c_str(), response.c_str());
    }

    for (int i = 0; i < 3 && !ret; i++) {
      int second = int(pow(2, i));
      std::this_thread::sleep_for(std::chrono::seconds(second));
      ret = HttpRequest::https_request_post_text(url, output, response);
      if (!ret) {
        Log::Error("task-output %s %s", task_info.task_id.c_str(), response.c_str());
      }
    }

	//一个命令已经成功执行完成，output应该清空，否则周期性命令输出会重叠
	output = "";
    if (!ret)
      return;

    Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
  }
}

void BaseTask::SendStoppedOutput() {
  if (HostFinder::getServerHost().empty()) {
    Log::Error("Get server host failed");
    return;
  }

  MutexLocker(&m_mutex) {
    int available_quota = task_info.output_info.log_quota - current;
    if (output.size() - current > available_quota)
      dropped = output.size() - current - available_quota;
    else
      dropped = 0;

    std::string url = ServiceProvide::GetStoppedOutputService();
    Log::Info("stop-done %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"taskId\": \"%s\", \"start\" : %lld, \"end\" : %lld, \
\"result\": \"killed\", \"dropped\": \"%lld\"}}, content:\n%s",
task_info.task_id.c_str(), url.c_str(),
task_info.task_id.c_str(), start_time, end_time,
dropped, output.c_str());

    if (current + dropped > 0) {
      output.erase(0, current + dropped);
    }

    char param[512];
    sprintf(param, "?taskId=%s&start=%lld&end=%lld&dropped=%d&result=killed",
      task_info.task_id.c_str(), start_time, end_time, dropped);
    url += param;
    std::string response;
    bool ret = HttpRequest::https_request_post_text(url, output, response);
    if (!ret) {
      Log::Error("task-output %s %s",
        task_info.task_id.c_str(), response.c_str());
    }

    for (int i = 0; i < 3 && !ret; i++) {
      int second = int(pow(2, i));
      std::this_thread::sleep_for(std::chrono::seconds(second));
      ret = HttpRequest::https_request_post_text(url, output, response);
      if (!ret) {
        Log::Error("task-output %s %s", task_info.task_id.c_str(), response.c_str());
      }
    }

    if (!ret)
      return;

    Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
  }
}

void BaseTask::SendTimeoutOutput() {
  if (HostFinder::getServerHost().empty()) {
    Log::Error("Get server host failed");
    return;
  }

  MutexLocker(&m_mutex) {
    int available_quota = task_info.output_info.log_quota - current;
    if (output.size() - current > available_quota)
      dropped = output.size() - current - available_quota;
    else
      dropped = 0;
    std::string url = ServiceProvide::GetTimeoutOutputService();
    Log::Info("timeout-task %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"taskId\": \"%s\", \"start\" : %lld, \"end\" : %lld, \
\"dropped\": \"%lld\"}}, content:\n%s",
task_info.task_id.c_str(), url.c_str(),
task_info.task_id.c_str(), start_time, end_time,
dropped, output.c_str());

    if (current + dropped > 0) {
      output.erase(0, current + dropped);
    }

    char param[512];
    sprintf(param, "?taskId=%s&start=%lld&end=%lld&dropped=%d",
      task_info.task_id.c_str(), start_time, end_time, dropped);
    url += param;
    std::string response;
    bool ret = HttpRequest::https_request_post_text(url, output, response);
    if (!ret) {
      Log::Error("task-output %s %s",
        task_info.task_id.c_str(), response.c_str());
    }

    for (int i = 0; i < 3 && !ret; i++) {
      int second = int(pow(2, i));
      std::this_thread::sleep_for(std::chrono::seconds(second));
      ret = HttpRequest::https_request_post_text(url, output, response);
      if (!ret) {
        Log::Error("task-output %s %s", task_info.task_id.c_str(), response.c_str());
      }
    }

    if (!ret)
      return;

    Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
  }
}

void BaseTask::SendErrorOutput() {
  if (HostFinder::getServerHost().empty()) {
    Log::Error("BaseTask::SendOutput, Get server host failed");
    return;
  }

  MutexLocker(&m_mutex) {

  }int available_quota = task_info.output_info.log_quota - current;
  if (output.size() - current > available_quota)
    dropped = output.size() - current - available_quota;
  else
    dropped = 0;

  std::string url = ServiceProvide::GetErrorOutputService();
  Log::Info("error-task %s {\"method\": \"POST\", \"url\": \"%s\", \
\"parameters\": {\"taskId\": \"%s\", \"start\" : %lld, \"end\" : %lld, \
\"exitCode\" : %d, \"dropped\": \"%lld\"}}, content:\n%s",
      task_info.task_id.c_str(), url.c_str(),
      task_info.task_id.c_str(), start_time, end_time,
      exit_code, dropped, output.c_str());

  if (current + dropped > 0) {
    output.erase(0, current + dropped);
  }

  char param[512];
  sprintf(param, "?taskId=%s&start=%lld&end=%lld&exitCode=%d&dropped=%d",
    task_info.task_id.c_str(), start_time, end_time, exit_code, dropped);
  url += param;
  std::string response;
  bool ret = HttpRequest::https_request_post_text(url, output, response);
  if (!ret) {
    Log::Error("task-output %s %s",
      task_info.task_id.c_str(), response.c_str());
  }

  for (int i = 0; i < 3 && !ret; i++) {
    int second = int(pow(2, i));
    std::this_thread::sleep_for(std::chrono::seconds(second));
    ret = HttpRequest::https_request_post_text(url, output, response);
    if (!ret) {
      Log::Error("task-output %s %s", task_info.task_id.c_str(), response.c_str());
    }
  }

  if (!ret)
    return;

  Log::Info("task-output %s %s", task_info.task_id.c_str(), response.c_str());
}

void BaseTask::DeleteOutputTimer() {
  if (output_timer) {
    Singleton<TimerManager>::I().deleteTimer(output_timer);
    output_timer = nullptr;
  }
}

}  // namespace task_engine
