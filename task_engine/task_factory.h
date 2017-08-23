// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_TASK_FACTORY_H_
#define CLIENT_TASK_ENGINE_TASK_FACTORY_H_

#include "./task.h"

#include <string>
#include <map>
#include <mutex>
namespace task_engine {
class TaskFactory {
 public:
  TaskFactory();
  Task* CreateTask(TaskInfo info);
  bool RemoveTask(std::string id);
  Task* GetTask(std::string id);
 private:
  std::map<std::string, Task*> task_maps;
  std::mutex mtx;
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_TASK_FACTORY_H_
