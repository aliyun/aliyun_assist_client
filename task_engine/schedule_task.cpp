// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./schedule_task.h"

#include <map>
#include <utility>
#include <string>
#include <vector>
#include <thread>

#include "./task_factory.h"
#include "./fetch_task.h"
#include "utils/singleton.h"
#include "utils/Log.h"
#include "utils/Encode.h"
#include "plugin/timer_manager.h"

namespace task_engine {
void Execute(void* context) {
  Log::Info("begin to execute task in thread");
  Task* task = reinterpret_cast<Task*>(context);
  if(!task) {
    Log::Error("task is nullptr");
    return;
  }
  task->Run();
  Log::Info("task after running");
  task->ReportOutput();
#if !defined(TEST_MODE)
  if (!task->IsPeriod()) {
    Singleton<TaskFactory>::I().RemoveTask(task->GetTaskInfo().task_id);
  }
#endif
}

void period_task_callback(void * context) {
  Log::Info("begin to execute period task in time thread");
  Task* task = reinterpret_cast<Task*>(context);
  std::thread t1(Execute, task);
  t1.detach();
}

TaskSchedule::TaskSchedule() {
}

#if defined(TEST_MODE)
void TaskSchedule::TestFetch(std::string info) {
  std::vector<TaskInfo> tasks;
  task_engine::TaskFetch task_fetch;
  Encoder encoder;
  char* pencodedata = encoder.B64Decode(
      (const unsigned char *)info.c_str(), info.size());
  task_fetch.TestFetchTasks(pencodedata, tasks);

  for (size_t i = 0; i < tasks.size(); i++) {
    Schedule(tasks[i]);
  }
}
#endif

void TaskSchedule::FetchPeriodTask() {
  std::vector<TaskInfo> tasks;
  task_engine::TaskFetch task_fetch;
  task_fetch.FetchPeriodTasks(tasks);

  for (size_t i = 0; i < tasks.size(); i++) {
    Schedule(tasks[i]);
  }
}

void TaskSchedule::Fetch() {
  std::vector<TaskInfo> tasks;
  std::vector<TaskInfo> canceled_tasks;
  task_engine::TaskFetch task_fetch;
  task_fetch.FetchTasks(tasks);
  task_fetch.FetchCancledTasks(canceled_tasks);

  for (size_t i = 0; i < tasks.size(); i++) {
    Schedule(tasks[i]);
  }

  for (size_t i = 0; i < canceled_tasks.size(); i++) {
    Cancel(canceled_tasks[i]);
  }
}

Task* TaskSchedule::Schedule(TaskInfo task_info) {
  Task* task = Singleton<TaskFactory>::I().CreateTask(task_info);
  if (!task) {
    Log::Error("Invalid task taskid:%s", task_info.task_id.c_str());
    return nullptr;
  }
  if (task->IsPeriod()) {
    if (period_tasks_.find(task_info.task_id) == period_tasks_.end()) {
      std::lock_guard<std::mutex> lck(mtx_);
      void* time_id = Singleton<TimerManager>::I().CreateTimer(
          &period_task_callback,
          reinterpret_cast<void*>(task), task_info.cronat);
      period_tasks_.insert(std::pair<std::string, void*>(
          task_info.task_id, time_id));
    }
  } else {
    std::thread t1(Execute, task);
    t1.detach();
  }
  return task;
}

void TaskSchedule::Cancel(TaskInfo task_info) {
  Log::Error("cancel task taskid:%s", task_info.task_id.c_str());
  Task* task = Singleton<TaskFactory>::I().GetTask(task_info.task_id);
  if (task) {
    if (task->IsPeriod()) {
      task->ReportStatus("stopped", task_info.instance_id);
      std::lock_guard<std::mutex> lck(mtx_);
      std::map<std::string, void*>::iterator iter =
          period_tasks_.find(task_info.task_id);
      if (iter != period_tasks_.end()) {
        Singleton<TimerManager>::I().DeleteTimer(iter->second);
      } 
    } else {
      task->Cancel();
    }
  }
}
}  // namespace task_engine
