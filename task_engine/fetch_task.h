// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_FETCH_TASK_H_
#define CLIENT_TASK_ENGINE_FETCH_TASK_H_
#include <vector>

#include "base_task.h"


namespace task_engine {
class TaskFetch {
 public:
  TaskFetch();
  void FetchTasks(std::vector<TaskInfo>& task_info);
  void FetchCancledTasks(std::vector<TaskInfo>& task_info);
  //void FetchPeriodTasks(std::vector<TaskInfo>& task_info);
#if defined(TEST_MODE)
  void TestFetchTasks(std::string res, std::vector<TaskInfo>& task_info);
#endif
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_FETCH_TASK_H_
