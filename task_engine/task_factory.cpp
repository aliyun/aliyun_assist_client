// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "./task_factory.h"

#include <utility>
#include <map>
#include <string>

#include "plugin/run_batscript.h"
#include "plugin/run_shellscript.h"
#include "plugin/bad_script.h"
#include "plugin/run_powshellscript.h"
#include "utils/Log.h"
#include "ccronexpr/ccronexpr.h"

namespace task_engine {
TaskFactory::TaskFactory() {
}

BaseTask* TaskFactory::CreateTask(RunTaskInfo& info) {
    BaseTask* task = nullptr;

	if ( !info.cronat.empty() ) {
		const char* err  = nullptr;
		cron_expr*  expr = cron_parse_expr(info.cronat.c_str(), &err);
		if (expr) {
			cron_expr_free(expr);
		}
		if ( err ) {
            Log::Error("invalid cron: %s, error: %s", info.cronat.c_str(), err);
            BadTask(info).SendInvalidTask("cron", info.cronat);
			return nullptr;
		}
	}
	
#if defined(_WIN32)
	if (!info.command_type.compare("RunPowerShellScript")) {
		task = new RunPowerShellTask(info);
	}
	if (!info.command_type.compare("RunBatScript")) {
		task = new RunBatTask(info);
	}
#else
	if (!info.command_type.compare("RunShellScript")) {
		task = new RunShellScriptTask(info);
	}
#endif

  if ( task ) {
	return  task;
  } 

  BadTask(info).SendInvalidTask("type", info.command_type);
  return task;
}

void TaskFactory::DeleteTask(BaseTask* info) {
	delete info;
}


}  // namespace task_engine
