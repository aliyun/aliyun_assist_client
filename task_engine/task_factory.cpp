// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "./task_factory.h"

#include <utility>
#include <map>
#include <string>

#include "plugin/install_package.h"
#include "plugin/run_batscript.h"
#include "plugin/run_shellscript.h"
#include "plugin/run_powshellscript.h"
#include "plugin/update_aliyunggent.h"
#include "utils/Log.h"

namespace task_engine {
TaskFactory::TaskFactory() {
}

Task* TaskFactory::CreateTask(TaskInfo info) {
  Task* task = nullptr;
  if (!info.command_id.compare("InstallPackage")) {
    task = new InsatllPackageTask(info);
  } else if (!info.command_id.compare("RunPowserShellScript")) {
#if defined(_WIN32)
    task = new RunPowserShellTask(info);
#endif
  } else if (!info.command_id.compare("RunBatScript")) {
#if defined(_WIN32)
    task = new RunBatTask(info);
#endif
  } else if (!info.command_id.compare("UpdateAgent")) {
    task = new UpdateAliyunAgentTask(info);
  } else if (!info.command_id.compare("RunShellScript")) {
#if !defined(_WIN32)
   task = new RunShellScriptTask(info);
#endif
 }
  if (task) {
    std::lock_guard<std::mutex> lck(mtx);
    task_maps.insert(std::pair<std::string, Task*>(info.task_id, task));
  } else {
    Log::Error("TaskFactory::CreateTask eror taskid:%s",
        info.task_id.c_str());
  }
  return task;
}

bool TaskFactory::RemoveTask(std::string id) {
  std::lock_guard<std::mutex> lck(mtx);
  std::map<std::string, Task*>::iterator iter;
  iter = task_maps.find(id);
  if (iter != task_maps.end()) {
    delete iter->second;
    task_maps.erase(iter);
  }
  return true;
}

Task* TaskFactory::GetTask(std::string id) {
  std::lock_guard<std::mutex> lck(mtx);
  Task* task = nullptr;
  std::map<std::string, Task*>::iterator iter;
  iter = task_maps.find(id);
  if (iter != task_maps.end()) {
    task = iter->second;
  }
  return task;
}
}  // namespace task_engine
