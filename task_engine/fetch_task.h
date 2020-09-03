// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_FETCH_TASK_H_
#define CLIENT_TASK_ENGINE_FETCH_TASK_H_
#include <vector>

#include "base_task.h"


namespace task_engine {
class TaskFetch {
 public:
  TaskFetch();
  void FetchTaskList(std::vector<task_engine::StopTaskInfo>& stop_task_info,
      std::vector<task_engine::RunTaskInfo>& run_task_info,
	  std::vector<task_engine::SendFile>& sendfile_task_info,
      std::string reason);
#if defined(TEST_MODE)
  //void TestFetchTasks(std::string res, std::vector<TaskInfo>& task_info);
#endif
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_FETCH_TASK_H_
