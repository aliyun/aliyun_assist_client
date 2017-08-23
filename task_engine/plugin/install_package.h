// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_PLUGIN_INSTALL_PACKAGE_H_
#define CLIENT_TASK_ENGINE_PLUGIN_INSTALL_PACKAGE_H_

#include "../task.h"

namespace task_engine {
class InsatllPackageTask : public Task {
 public:
  explicit InsatllPackageTask(TaskInfo info);
  void Run();
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_INSTALL_PACKAGE_H_
