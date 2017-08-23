// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./run_shellscript.h"

#include <string>

#include "utils/AssistPath.h"
#include "utils/TimeTool.h"
#include "utils/SubProcess.h"

namespace task_engine {
RunShellScriptTask::RunShellScriptTask(TaskInfo info) : Task(info) {
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
  string out;
  string cmd = task_info_.content;
  sub_process_.set_cmd(cmd);
  sub_process_.Execute(task_output_, err_code_);
}
}  // namespace task_engine
