// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./run_shellscript.h"

#include <string>
#include <mutex>

#include "utils/AssistPath.h"
#include "utils/process.h"
#include "utils/Log.h"

namespace task_engine {
RunShellScriptTask::RunShellScriptTask(TaskInfo info) : BaseTask(info) {
}

bool RunShellScriptTask::BuildScript(string fileName, string content) {
  
  
  FILE *fp = fopen(fileName.c_str(), "a+");
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
  string  dir = task_info.working_dir;
  int timeout = atoi(task_info.time_out.c_str());
  DoWork(cmd, dir, timeout);
}
}  // namespace task_engine
