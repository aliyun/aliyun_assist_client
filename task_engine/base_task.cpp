// Copyright (c) 2017-2018 Alibaba Group Holding Limited.


#include <string>

#include "base_task.h"
#include "utils/host_finder.h"
#include "utils/http_request.h"
#include "utils/encoder.h"
#include "utils/Log.h"
#include "utils/service_provide.h"
#include "utils/singleton.h"
#include "json11/json11.h"

#if !defined(_WIN32)
#include<sys/types.h>
#include<signal.h>
#include <sys/wait.h>
#include <unistd.h>
#else
#include <windows.h>
#endif



namespace task_engine {



BaseTask::BaseTask(TaskInfo info)  {
  task_info = info;
  timer = nullptr;
  canceled = false;
}





void BaseTask::DoWork(std::string cmd, std::string dir, int timeout) {
	std::string output;
	auto callback = [&output]( const char* buf, size_t len ) {
		 output = output + string(buf).substr(0, len);
		if ( output.size() > MAX_TASK_OUTPUT) {
			 output.erase(0, output.size() - MAX_TASK_OUTPUT);
		}
	};
	int code;
	Process::RunResult result = Process(cmd, dir).syncRun(timeout, callback, callback, &code);
	if ( result == Process::timeout ) {
		ReportTimeout(output);
	}
	else if( result == Process::fail ) {
		ReportStatus("failed");
	}
	else {
		ReportOutput(output,code);
	}
};


void BaseTask::Cancel() {
  
  Log::Info("cancel the task:%s", task_info.task_id.c_str());
  canceled = true;
  ReportStatus("stopped");
}




void BaseTask::ReportStatus(std::string status) {

  Log::Info("report taskid:%s status:%s ", task_info.task_id.c_str(), status.c_str() );
  std::string response;
  std::string input;

  json11::Json json = json11::Json::object{
	  {"taskStatus",status },
	  {"taskID",task_info.task_id},
  };
  
  if (HostFinder::getServerHost().empty()) {
	  return;
  }

  std::string url = ServiceProvide::GetReportTaskStatusService();
  bool ret = HttpRequest::https_request_post(url, json.dump(), response);

  for (int i = 0; i < 3 && !ret; i++) {
	  std::this_thread::sleep_for(std::chrono::seconds(3));
	  ret = HttpRequest::https_request_post(url, json.dump(), response);
  }
}



void BaseTask::ReportOutput(std::string output, int exitcode) {

  if ( HostFinder::getServerHost().empty() ) {
	return;
  }

  if ( output.size() > MAX_TASK_OUTPUT ) {
	   output.erase(0, output.size() - MAX_TASK_OUTPUT);
  }

  std::string status_ = "finished";
  std::string response;
  Encoder     encoder;
#if defined(_WIN32)
  std::string utf8_data = encoder.Gbk2Utf(output);
  char* pencodedata = encoder.B64Encode(
              (const unsigned char *)utf8_data.c_str(), utf8_data.size());
#else
  char* pencodedata = encoder.B64Encode(
               (const unsigned char *)output.c_str(),output.size());
#endif
  
  json11::Json json = json11::Json::object{
	 { "taskID",task_info.task_id },
	 { "taskStatus",status_ },
	 { "taskOutput",json11::Json::object{  {"taskInstanceOutput",pencodedata},{"errNo",exitcode } } }
  };

  free(pencodedata);

  std::string url = ServiceProvide::GetReportTaskOutputService();
  bool ret = HttpRequest::https_request_post(url, json.dump(), response);

  Log::Info("report taskid:%s output:%s exitcode:%d response:%s",
      task_info.task_id.c_str(), output.c_str(),exitcode, response.c_str());

  for (int i = 0; i < 3 && !ret; i++) {
	  std::this_thread::sleep_for(std::chrono::seconds(3));
	  ret = HttpRequest::https_request_post(url, json.dump(), response);
  }
}


	
void BaseTask::ReportTimeout(std::string output) {
  
  Log::Info("Report timeout");
  if (HostFinder::getServerHost().empty()) {
	  return;
  }

  if (output.size() > MAX_TASK_OUTPUT) {
	  output.erase(0, output.size() - MAX_TASK_OUTPUT);
  }

  std::string response;
  std::string input;

  Encoder encoder;
#if defined(_WIN32)
	std::string utf8_data = encoder.Gbk2Utf(output);
	char* pencodedata = encoder.B64Encode(
	            (const unsigned char *)utf8_data.c_str(),utf8_data.size());
#else
	char* pencodedata = encoder.B64Encode(
	             (const unsigned char *)output.c_str(),output.size());
#endif
 
 
  json11::Json json = json11::Json::object{
	  {"taskID",task_info.task_id},
	  {"taskStatus","failed"},
	  {"taskOutput",json11::Json::object{{"taskInstanceOutput",pencodedata},{"errNo",-1}} }
  };
  
  free(pencodedata);
  std::string url = ServiceProvide::GetReportTaskOutputService();
  bool ret = HttpRequest::https_request_post(url, json.dump(), response);

  Log::Info("Report taskid:%s task_output:%s error_code:%d %s:response",
      task_info.task_id.c_str(), output.c_str(),
      -1, response.c_str());

  for (int i = 0; i < 3 && !ret; i++) {
	  std::this_thread::sleep_for(std::chrono::seconds(3));
	  ret = HttpRequest::https_request_post(url, json.dump(), response);
  }
   
}

}  // namespace task_engine
