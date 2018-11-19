// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_SCHEDULE_TASK_H_
#define CLIENT_TASK_ENGINE_SCHEDULE_TASK_H_

#include <string>
#include <map>
#include <mutex>

#include "./task.h"

namespace task_engine {
class TaskSchedule {
 public:
  TaskSchedule();
  void Cancel(TaskInfo task_info);
  int Fetch(bool from_kick=false);
#if defined(TEST_MODE)
  void TestFetch(std::string info);
#endif
  void FetchPeriodTask();
  Task* Schedule(TaskInfo task_info);
 private:
  std::map<std::string, void*> period_tasks_;
  std::mutex mtx_;
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_SCHEDULE_TASK_H_
