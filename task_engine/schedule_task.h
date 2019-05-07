// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_SCHEDULE_TASK_H_
#define CLIENT_TASK_ENGINE_SCHEDULE_TASK_H_

#include <string>
#include <map>
#include <mutex>

#include "base_task.h"

namespace task_engine {
class TaskSchedule {
 public:
  TaskSchedule();

  void		Cancel(TaskInfo task_info);
  int		Fetch(bool from_kick=false);
  void		FetchPeriodTask();
  void      Schedule(TaskInfo task_info);
private:
  void DispatchTask(BaseTask* task);
  void Execute(BaseTask* task);

private:
  std::map<std::string, BaseTask*> m_tasklist;
  std::mutex m_mutex;
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_SCHEDULE_TASK_H_
