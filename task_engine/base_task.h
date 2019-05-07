// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_TASK_H_
#define CLIENT_TASK_ENGINE_TASK_H_

#include <string>
#include <vector>
#include "utils/process.h"

#define MAX_TASK_OUTPUT 12*1024
namespace task_engine {



struct TaskInfo {
  std::string task_id;
  std::string command_type;
  std::string content;
  std::string params;
  std::string cronat;
  std::string working_dir;
  std::string time_out;
};

class BaseTask {
 public:
  virtual ~BaseTask() {};
  explicit BaseTask(TaskInfo info);
  virtual void Run() = 0;

  void Cancel();
  void ReportStatus(std::string status);
  void ReportOutput(std::string output,int exitcode);
  void ReportTimeout(std::string output);
  
  void*    timer;
  bool     canceled;
  TaskInfo task_info;

 protected:
   void DoWork(std::string cmd, std::string dir,int timeout);
  
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_TASK_H_
