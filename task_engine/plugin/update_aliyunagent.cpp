// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./update_aliyunggent.h"

#include "utils/SubProcess.h"

namespace task_engine {
UpdateAliyunAgentTask::UpdateAliyunAgentTask(TaskInfo info) : Task(info) {
}

void UpdateAliyunAgentTask::Run() {
  sub_process_.set_cmd("aliyun_assist_update.exe --check_update");
  sub_process_.Execute(task_output_, err_code_);
}
}  // namespace task_engine
