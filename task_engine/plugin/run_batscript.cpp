// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./run_batscript.h"

#include <string>
#include <functional>
#include <mutex>

#include "utils/AssistPath.h"
#include "utils/process.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"


namespace task_engine {
RunBatTask::RunBatTask(TaskInfo info) : BaseTask(info) {
}

bool RunBatTask::BuildScript(string fileName, string content) {

  if ( FileUtils::fileExists(fileName.c_str() ) ) {
	  return true;
  };

  FILE *fp = fopen(fileName.c_str(), "a+");
  if (!fp) {
    return false;
  }
  std::string echo_off = "@echo off\n";
  fwrite(echo_off.c_str(), echo_off.size(), 1, fp);
  fwrite(content.c_str(), content.size(), 1, fp);
  fclose(fp);
  fp = nullptr;
  return true;
}

void RunBatTask::Run() {
 
  AssistPath assistPath("");
  string scriptPath = assistPath.GetWorkPath("script");
 
  string filename = scriptPath + "\\" + task_info.task_id + ".bat";
  BuildScript(filename, task_info.content);

  
  string cmd  = filename;
  string  dir = task_info.working_dir;
  int timeout = atoi(task_info.time_out.c_str());
  DoWork(cmd, dir, timeout);
  //sub_process_.set_cmd(cmd);
  //sub_process_.Execute(task_output_, err_code_);
}
}  // namespace task_engine
