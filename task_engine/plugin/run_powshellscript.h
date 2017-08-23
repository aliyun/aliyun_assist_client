// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_RUN_POWSHELLSCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_RUN_POWSHELLSCRIPT_H_
#include <string>
#include "../task.h"

using std::string;

namespace task_engine {
class RunPowserShellTask : public Task {
 public:
  explicit RunPowserShellTask(TaskInfo info);
  void Run();
 private:
  bool BuildScript(string fileName, string content);
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_RUN_POWSHELLSCRIPT_H_
