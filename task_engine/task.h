// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_TASK_H_
#define CLIENT_TASK_ENGINE_TASK_H_

#include <string>
#include <vector>
#include "utils/SubProcess.h"
namespace task_engine {
struct TaskInfo {
  std::string task_id;
  std::string instance_id;
  std::string command_id;
  std::string content;
  std::string params;
  std::string cronat;
  std::string working_dir;
  std::string time_out;
};

class Task {
 public:
  Task();
  explicit Task(TaskInfo info);
  virtual void Run();
  void Cancel();
  void ReportStatus(std::string status, std::string instance_id="");
  void ReportOutput();
  void ReportTimeout();
  bool IsPeriod() { return is_period_; }
  std::string GetOutput() { return task_output_; }
  TaskInfo GetTaskInfo() { return task_info_; }
  void CheckTimeout();
 protected:
  std::string task_output_;
  long err_code_;
  std::string status_;
  TaskInfo task_info_;
  SubProcess sub_process_;
  bool is_period_;
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_TASK_H_
