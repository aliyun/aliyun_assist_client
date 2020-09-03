// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "debug_script.h"

#include <string>
#include "utils/process.h"
#include "utils/Log.h"
#include "utils/AssistPath.h"
#include "utils/service_provide.h"

namespace task_engine {
DebugTask::DebugTask(){
}

void  DebugTask::RunSystemNetCheck() {
#if defined WIN32
  return;
#endif

  std::string url = ServiceProvide::GetConnectDetectService();
  std::string commond = "curl -s " + url;
  std::string process_output;

  auto callback = [&]( const char* buf, size_t len ) {
    if (!buf || 0 == len)
    {
     return;
    }
    process_output += buf;
  };

  int exit_code = 0;
	Process::RunResult result = Process(commond, "").syncRun(10, callback, callback, &exit_code);
  Log::Info("check system network: %s", process_output.c_str());

  if(process_output == "ok") {
    RunRetartAssist();
  }
}

void DebugTask::RunRetartAssist() {
  Log::Info("internal error, try to restart assist");
  std::string commond = R"(/etc/init.d/aliyun-service restart
          /sbin/initctl restart aliyun-service
             systemctl restart aliyun.service)";
  Process(commond).syncRun(10);
}

}  // namespace task_engine
