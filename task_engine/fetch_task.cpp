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
void parse_task_info(std::string response,
    std::vector<task_engine::TaskInfo>& task_info) {
 
  try {
	  string errinfo;
	  auto json = json11::Json::parse(response, errinfo);
	  if (errinfo != "") {
		  Log::Error("invalid json format");
		  return;
	  }
    
	  for ( auto &it : json.array_items() ) {
		  task_engine::TaskInfo info;
		 
		  auto taskjson = it["taskItem"];
		  if ( taskjson.is_null() )
			  continue;

		  info.command_type = taskjson["type"].string_value();
		  info.task_id = taskjson["taskID"].string_value();
		  std::string content = taskjson["commandContent"].string_value();
		  Encoder encode;
		  if ( !content.empty() ) {
			  info.content = reinterpret_cast<char *>(encode.B64Decode(
				  content.c_str(), content.size()));
		  }
		  info.working_dir = taskjson["workingDirectory"].string_value();
		  info.cronat = taskjson["cron"].string_value();
		  info.time_out = taskjson["timeOut"].string_value();
		  if ( info.time_out.empty() ) {
			  info.time_out = "3600";
		  }
		  task_info.push_back(info);
	  }

  } catch(...) {
    Log::Error("fetch task json is invalid");
  }
}
}  // namespace

namespace task_engine {
TaskFetch::TaskFetch() {
}

void TaskFetch::FetchTasks(std::vector<TaskInfo>& task_info) {
 
  if (HostFinder::getServerHost().empty() ) {
	return;
  }
  std::string response;
  std::string url = ServiceProvide::GetFetchTaskService();
  HttpRequest::https_request_post(url, "", response);
  parse_task_info(response, task_info);
  Log::Info("Fetch %d Tasks response is: %s", task_info.size(), response.c_str());
}

#if defined(TEST_MODE)
void TaskFetch::TestFetchTasks(std::string res, std::vector<TaskInfo>& task_info) {
  parse_task_info(res, task_info);
}
#endif

void TaskFetch::FetchCancledTasks(std::vector<TaskInfo>& task_info) {
  std::string response;
  if (HostFinder::getServerHost().empty()) {
    return;
  }
  std::string url = ServiceProvide::GetFetchCanceledTaskService();
  HttpRequest::https_request_post(url, "", response);
  parse_task_info(response, task_info);
  Log::Info("Fetch_Cancled_Tasks:Fetch %d Tasks", task_info.size());
}

/*void TaskFetch::FetchPeriodTasks(std::vector<TaskInfo>& task_info) {
  std::string response;
  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = ServiceProvide::GetFetchPeriondTaskService();
  HttpRequest::https_request_post(url, "", response);
  parse_task_info(response, task_info);
  Log::Info("Fetch_Period_Tasks:Fetch %d Tasks", task_info.size());
}*/

}  // namespace task_engine
