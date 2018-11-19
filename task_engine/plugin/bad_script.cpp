// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./bad_script.h"

#include <string>

#include "utils/AssistPath.h"
#include "utils/TimeTool.h"
#include "utils/SubProcess.h"

namespace task_engine {
BadTask::BadTask(TaskInfo info) : Task(info) {
}

void BadTask::Run() {

}
}  // namespace task_engine
