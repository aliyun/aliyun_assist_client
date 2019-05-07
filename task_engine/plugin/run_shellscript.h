// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_RUNSHELLSCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_RUNSHELLSCRIPT_H_

#include <string>
#include "../base_task.h"

namespace task_engine {
class RunShellScriptTask : public BaseTask {
 public:
  explicit RunShellScriptTask(TaskInfo info);
  virtual  ~RunShellScriptTask() {};
  void Run();
 private:
  bool BuildScript(string fileName, std::string content);
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_RUNSHELLSCRIPT_H_
