// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_DEBUG_SCRIPT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_DEBUG_SCRIPT_H_

#include <string>

namespace task_engine {
class DebugTask {
 public:
  explicit DebugTask();
  virtual ~DebugTask() {};
  void RunSystemNetCheck();
  void RunRetartAssist();
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_DEBUG_SCRIPT_H_
