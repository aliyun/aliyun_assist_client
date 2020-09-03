// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./run_powshellscript.h"

#include <string>
#include <mutex>

#include "./run_batscript.h"
#include "utils/AssistPath.h"
#include "utils/process.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"

namespace task_engine {
RunPowerShellTask::RunPowerShellTask(RunTaskInfo info) : BaseTask(info) {
}

bool RunPowerShellTask::BuildScript(string fileName, string content) {
 
  if (FileUtils::fileExists(fileName.c_str())) {
	 return false;
  }

  FILE* fp = fopen(fileName.c_str(), "w+");
  if (!fp) {
    return false;
  }
  fwrite(content.c_str(), content.size(), 1, fp);
  fclose(fp);
  fp = NULL;
  return true;
}

void RunPowerShellTask::Run() {

  AssistPath assistPath("../");
  string scriptPath = assistPath.GetWorkPath("script");
  string filename = scriptPath + "\\"  + task_info.task_id + ".ps1";
  if (BuildScript(filename, task_info.content) == false) {
    if (task_info.cronat.empty()) {
      Log::Info("duplicate task ignore:%s", task_info.task_id.c_str());
      return;
    }
  }

  
  Process("powershell.exe Set-ExecutionPolicy RemoteSigned")
	  .syncRun(10);

  string  cmd = "powershell -file \"" + filename + "\"";
  string  dir = task_info.working_dir;
  int timeout = atoi( task_info.time_out.c_str() );
  DoWork(cmd, dir, timeout);

}




}  // namespace task_engine
