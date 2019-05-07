// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_RUN_POWSHELLSCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_RUN_POWSHELLSCRIPT_H_
#include <string>
#include "../base_task.h"

using std::string;

namespace task_engine {
class RunPowerShellTask : public BaseTask {
 public:
  explicit RunPowerShellTask(TaskInfo info);
  virtual  ~RunPowerShellTask() {};
  void Run();
 private:
  bool BuildScript(string fileName, string content);
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_RUN_POWSHELLSCRIPT_H_
