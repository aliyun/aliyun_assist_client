// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_RUN_BATSCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_RUN_BATSCRIPT_H_

#include <string>
#include "../base_task.h"

namespace task_engine {
class RunBatTask : public BaseTask {
 public:
  explicit RunBatTask(TaskInfo info);
  virtual ~RunBatTask() {};
  void Run();
 private:
  bool BuildScript(string fileName, std::string content);
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_RUN_BATSCRIPT_H_
