// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "./install_package.h"

#include <string>

#include "jsoncpp/json.h"
#include "utils/SubProcess.h"

using std::string;

namespace task_engine {

InsatllPackageTask::InsatllPackageTask(TaskInfo info) : Task(info) {
}

void InsatllPackageTask::Run() {
  Json::Value  jsonRoot;
  Json::Reader reader;
  if (!reader.parse(task_info_.content, jsonRoot)) {
    return;
  }

  string  package = jsonRoot["package"].asString();
  string  version = jsonRoot["package_version"].asString();
  string  arch = jsonRoot["arch"].asString();

  string cmd = string("aliyun_installer.exe --install ") +
    "--package=" + package +
    "--package_version=" + package +
    "--arch=" + arch;

  sub_process_.set_cmd(cmd);
  sub_process_.Execute(task_output_, err_code_);
}

}  // namespace task_engine

