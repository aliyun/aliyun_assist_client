// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./update_aliyunggent.h"

#include "utils/SubProcess.h"

namespace task_engine {
UpdateAliyunAgentTask::UpdateAliyunAgentTask(TaskInfo info) : Task(info) {
}

void UpdateAliyunAgentTask::Run() {
  sub_process_.set_cmd(task_info_.content);
  sub_process_.RunModule("aliyun_assist_update");
}
}  // namespace task_engine
