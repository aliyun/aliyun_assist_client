// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_RUNSHELLSCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_RUNSHELLSCRIPT_H_

#include <string>
#include "../task.h"

namespace task_engine {
class RunShellScriptTask : public Task {
 public:
  explicit RunShellScriptTask(TaskInfo info);
  void Run();
 private:
  bool BuildScript(string fileName, std::string content);
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_RUNSHELLSCRIPT_H_
