// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_PLUGIN_UPDATE_ALIYUNGGENT_H_
#define CLIENT_TASK_ENGINE_PLUGIN_UPDATE_ALIYUNGGENT_H_

#include "../task.h"

namespace task_engine {
class UpdateAliyunAgentTask : public Task {
 public:
  explicit UpdateAliyunAgentTask(TaskInfo info);
  void Run();
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_UPDATE_ALIYUNGGENT_H_
