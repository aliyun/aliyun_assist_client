// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./run_batscript.h"

#include <string>

#include "utils/AssistPath.h"
#include "utils/TimeTool.h"
#include "utils/SubProcess.h"

namespace task_engine {
RunBatTask::RunBatTask(TaskInfo info) : Task(info) {
}

bool RunBatTask::BuildScript(string fileName, string content) {
  FILE *fp = fopen(fileName.c_str(), "a+");
  if (!fp) {
    return false;
  }
  fwrite(content.c_str(), content.size(), 1, fp);
  fclose(fp);
  fp = nullptr;
  return true;
}

void RunBatTask::Run() {
  AssistPath assistPath("");
  string scriptPath = assistPath.GetWorkPath("script");
  string time = Time::GetLocalTime();
  string filename = scriptPath + "\\" + time + task_info_.task_id + ".bat";
  BuildScript(filename, task_info_.content);

  string out;
  string cmd = filename;
  sub_process_.set_cmd(cmd);
  sub_process_.Execute(task_output_, err_code_);
}
}  // namespace task_engine
