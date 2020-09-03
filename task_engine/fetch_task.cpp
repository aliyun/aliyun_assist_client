// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./fetch_task.h"

#include <vector>
#include <string>
#include <math.h>

#include "utils/http_request.h"
#include "utils/service_provide.h"
#include "utils/host_finder.h"
#include "utils/encoder.h"
#include "utils/Log.h"
#include "utils/AssistPath.h"
#include "json11/json11.h"
#include "md5/md5.h"
#include "utils/CStringUtil.h"
#include "plugin/bad_script.h"

#include "plugin/debug_script.h"

#include "utils/SystemInfo.h"


namespace {

void ParseSendFileInfo(json11::Json json,
	std::vector<task_engine::SendFile>& send_file_info)
{
	try {
		for (auto &it : json.array_items()) {
			task_engine::SendFile info;

			auto taskjson = it["task"];
			if (taskjson.is_null()) {
				Log::Error("sendfile task json is null");
				continue;
			}

			info.name = taskjson["name"].string_value();
			//info.contentType = taskjson["contentType"].string_value();
			info.content = taskjson["content"].string_value();
			info.signature = taskjson["signature"].string_value();
			info.invokeId = taskjson["taskID"].string_value();
			info.timeout = taskjson["timeout"].string_value();
			info.destination = taskjson["destination"].string_value();
			info.fileType = taskjson["fileType"].string_value();
			info.owner = taskjson["owner"].string_value();
			info.group = taskjson["group"].string_value();
			info.mode = taskjson["mode"].string_value();
			info.overwrite = taskjson["overwrite"].bool_value();
			send_file_info.push_back(info);
		}
	}
	catch (...) {
		Log::Error("sendfile task list json is invalid");
	}
}

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
	  //增加命令签名校验功能
	  //std::string taskSignature = taskjson["taskSignature"].string_value();
	  //if (taskSignature.length() > 0){
		 // std::string content_md5 = md5(content);
		 // CStringUtil::ToLower(content_md5);
		 // CStringUtil::ToLower(taskSignature);
		 // //为什么要这么修改：服务端生成的MD5方式有点特殊，前导0被去掉了，导致服务端传递过来的MD5可能会比客户端短
		 // if (content_md5.find(taskSignature) == std::string::npos) {
			//  std::string value = taskSignature;
			//  value.append("/");
			//  value.append(content_md5);
			//  task_engine::BadTask(info).SendInvalidTask("taskSignature", value);
			//  Log::Error("invalid taskSignature,task_id = %s", info.task_id.c_str());
			//  continue;
		 // }
	  //}
	  

      Encoder encode;
      if (!content.empty()) {
        info.content = reinterpret_cast<char *>(encode.B64Decode(
          content.c_str(), content.size()));
      }
#ifdef _WIN32
	  LCID lcid = SystemInfo::GetWindowsDefaultLang();
	  if (lcid != 0x409) {//非英文环境才转换
		  info.content = CStringUtil::Utf8ToAscii(info.content);
	  }
#endif
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
  std::vector<task_engine::RunTaskInfo>& run_task_info,
  std::vector<task_engine::SendFile>& send_file_info) {

  try {
    string errinfo;
    auto json = json11::Json::parse(response, errinfo);
    if (errinfo != "") {
      Log::Error("invalid json format");
      return;
    }

    ParseStopTaskInfo(json["stop"], stop_task_info);
    ParseRunTaskInfo(json["run"], run_task_info);
	ParseSendFileInfo(json["file"], send_file_info);
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
	std::vector<task_engine::SendFile>& sendfile_task_info,
    std::string reason) {

  if (HostFinder::getServerHost().empty()) {
    return;
  }
  std::string response;
  std::string url = ServiceProvide::GetFetchTaskListService();
  Log::Info("fetch-task request {\"method\": \"POST\", \"url\" : \"%s\", \"parameters\" : {\"reason\": \"%s\"} }", url.c_str(), reason.c_str());

  url = url + "?reason=" + reason;
  bool ret = HttpRequest::https_request_post(url, "", response);
  if (!ret) {
    Log::Error("fetch-task response %s", response.c_str());
  }

  for (int i = 0; i < 3 && !ret; i++) {
    int second = int(pow(2, i));
    std::this_thread::sleep_for(std::chrono::seconds(second));
    ret = HttpRequest::https_request_post(url, "", response);
    if (!ret) {
      Log::Error("fetch-task response %s", response.c_str());
    }
  }

  if (!ret) {
    Log::Error("assist network is wrong");
    DebugTask task;
    task.RunSystemNetCheck();
    return;
  }


  Log::Info("fetch-task response %s", response.c_str());
  ParseTaskInfo(response, stop_task_info, run_task_info, sendfile_task_info);
}

}  // namespace task_engine
