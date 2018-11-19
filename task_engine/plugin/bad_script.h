// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_BAD_SCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_BAD_SCRIPT_H_

#include <string>
#include "../task.h"

namespace task_engine {
class BadTask : public Task {
 public:
  explicit BadTask(TaskInfo info);
  void Run();
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_BAD_SCRIPT_H_
