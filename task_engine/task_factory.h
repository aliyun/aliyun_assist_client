// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_TASK_FACTORY_H_
#define CLIENT_TASK_ENGINE_TASK_FACTORY_H_

#include "base_task.h"

#include <string>
#include <map>
#include <mutex>
namespace task_engine {
class TaskFactory {
 public:
  TaskFactory();
  BaseTask*  CreateTask(TaskInfo& info);
  void       DeleteTask(BaseTask* task);
 private:
  std::map<std::string, BaseTask*> task_history;
  std::mutex mtx;
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_TASK_FACTORY_H_
