// Copyright (c) 2017-2018 Alibaba Group Holding Limited
#ifndef CLIENT_TASK_ENGINE_TASK_H_
#define CLIENT_TASK_ENGINE_TASK_H_

#include <string>
#include <vector>
#include "utils/process.h"
#include "timer_manager.h"

#define MAX_TASK_OUTPUT 12*1024
namespace task_engine {
  struct BaseTaskInfo {
    BaseTaskInfo() : json_data(""),
        instance_id(""),
        command_type(""),
        task_id(""),
        command_id(""),
        time_out(""),
        output_info() {
    };

    virtual ~BaseTaskInfo() {};

    //Task meta data
    std::string json_data;
    std::string instance_id;
    std::string command_type;
    std::string task_id;
    std::string command_id;
    std::string time_out;

    struct OutputInfo {
      OutputInfo() : interval(0),
          log_quota(0),
          skip_empty(false),
          send_start(false){
      };
      int interval;
      int log_quota;
      bool skip_empty;
      bool send_start;
    } output_info;
  };

  struct StopTaskInfo : public BaseTaskInfo {
    StopTaskInfo() : BaseTaskInfo(),
      command_name(""),
      content(""),
      working_dir(""),
      args(""),
      cronat("") {
    };

    //Task meta data
    std::string command_name;
    std::string content;
    std::string working_dir;
    std::string args;
    std::string cronat;
    //std::string command_signature;
  };

  struct RunTaskInfo : public BaseTaskInfo {
    RunTaskInfo() : BaseTaskInfo(),
      command_name(""),
      content(""),
      working_dir(""),
      args(""),
      cronat("") {
    };

    //Task meta data
    std::string command_name;
    std::string content;
    std::string working_dir;
    std::string args;
    std::string cronat;
    //std::string command_signature;
  };

  struct SendFile
  {
	  std::string name;
	  //std::string contentType;
	  std::string content;
	  std::string signature;
	  std::string invokeId;
	  std::string timeout;
	  std::string destination;
	  std::string fileType;
	  std::string owner;
	  std::string group;
	  std::string mode;
	  bool overwrite;
  };

class BaseTask {
 public:
  virtual ~BaseTask() {
    DeleteOutputTimer();
  };
  explicit BaseTask(RunTaskInfo info);
  virtual void Run() = 0;

  void Cancel();
  void SendInvalidTask(std::string param, std::string value);
  void SendTaskStart();
  void SendRunningOutput();
  void SendFinishOutput();
  void SendStoppedOutput();
  void SendTimeoutOutput();
  void SendErrorOutput();
 
 public:
  void*    timer;
  bool     canceled;
  RunTaskInfo task_info;

 private:
   void DeleteOutputTimer();

 private:
  Timer* output_timer;
  std::mutex m_mutex;

  //Task status for run time
  std::string cmd;
  int64_t start_time;
  int64_t end_time;
  std::string output;
  std::string running_output;
  //bool truncated;
  int exit_code;

  //Used to record the output accepted by server
  int64_t received;
  int64_t accepted;
  int64_t current;
  int64_t dropped;

 protected:
   void DoWork(std::string cmd, std::string dir,int timeout);
  
};
}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_TASK_H_
