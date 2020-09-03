// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./run_shellscript.h"

#include <string>
#include <mutex>

#include "utils/AssistPath.h"
#include "utils/process.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"

namespace task_engine {
RunShellScriptTask::RunShellScriptTask(RunTaskInfo info) : BaseTask(info) {
}

bool RunShellScriptTask::BuildScript(string fileName, string content) {
  
  if ( FileUtils::fileExists(fileName.c_str()) ) {
	 return false;
  };
  
  FILE *fp = fopen(fileName.c_str(), "w+");
  if (!fp) {
    return false;
  }
  fwrite(content.c_str(), content.size(), 1, fp);
  fclose(fp);
  fp = NULL;
  return true;

}

void RunShellScriptTask::Run() {
  string cmd  = task_info.content;
  // just back up to local file 
  AssistPath assistPath("");
  string scriptPath = assistPath.GetScriptPath();
  string filename = scriptPath + "/"  + task_info.task_id + ".sh";

  if (BuildScript(filename, task_info.content) == false) {
    if (task_info.cronat.empty()) {
      Log::Info("duplicate task ignore:%s", task_info.task_id.c_str());
      return;
    }
  }
  Process("chmod +x " + filename).syncRun(10);

  string  dir = task_info.working_dir;
  int timeout = atoi(task_info.time_out.c_str());
  DoWork(filename, dir, timeout);
}
}  // namespace task_engine
