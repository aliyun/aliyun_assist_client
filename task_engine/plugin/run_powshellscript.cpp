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
RunPowerShellTask::RunPowerShellTask(TaskInfo info) : BaseTask(info) {
}

bool RunPowerShellTask::BuildScript(string fileName, string content) {
 
  if ( FileUtils::fileExists(fileName.c_str()) ) {
	 return true;
  };

  FILE* fp = fopen(fileName.c_str(), "a+");
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
  BuildScript(filename, task_info.content);

  
  Process("powershell.exe Set-ExecutionPolicy RemoteSigned")
	  .syncRun(10);

  string  cmd = "powershell -file \"" + filename + "\"";
  string  dir = task_info.working_dir;
  int timeout = atoi( task_info.time_out.c_str() );
  DoWork(cmd, dir, timeout);

}




}  // namespace task_engine
