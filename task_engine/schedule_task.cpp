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
#include "utils/encoder.h"
#include "utils/MutexLocker.h"
#include "timer_manager.h"
#include "utils/DirIterator.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/config.h"
#if !defined(_WIN32)
#include <pthread.h>
#include <unistd.h>
#else
#include <windows.h>
#endif
#include "sendfile.h"

namespace task_engine {

TaskSchedule::TaskSchedule() {
}

#if defined(TEST_MODE)
void TaskSchedule::TestFetch(std::string info) {
  std::vector<TaskInfo> tasks;
  task_engine::TaskFetch task_fetch;
  Encoder encoder;
  std::string encodedata = reinterpret_cast<char *>(encoder.B64Decode(info.c_str(), info.size()));
  task_fetch.TestFetchTasks(encodedata, tasks);

  for (size_t i = 0; i < tasks.size(); i++) {
    Schedule(tasks[i]);
  }
}
#endif

void TaskSchedule::CleanTasks() {
  AssistPath assistPath("");
  std::string dir = assistPath.GetScriptPath();

  std::string task_reserved_time_in_days;
  std::string task_reserved_num;

  task_reserved_time_in_days = AssistConfig::GetConfigValue("task_reserved_time_in_days", "30");
  task_reserved_num = AssistConfig::GetConfigValue("task_reserved_num", "1000");

  if (dir.empty() || !FileUtils::fileExists(dir.c_str())) {
    return;
  }
  int file_cnt = 0;
  DirIterator it_dir(dir.c_str());
  while (it_dir.next()) {
    std::string name = it_dir.fileName();
    if (name == "." || name == "..")
      continue;

    if (it_dir.isDir())
      continue;

    file_cnt++;
    std::string file_path = dir + Log::separator() + name;

    time_t currenttime = time(NULL);
    struct stat file_info;
    stat(file_path.c_str(), &file_info);
    double totalT = difftime(currenttime, file_info.st_ctime);
    if (totalT > atoi(task_reserved_time_in_days.c_str()) * 24 * 3600) {
      FileUtils::removeFile(file_path.c_str());
      Log::Info("clean history tasks:%s",  file_path.c_str());
    }
  }
  if(file_cnt > atoi(task_reserved_num.c_str())) {
    FileUtils::rmdirRecursive(dir.c_str());
    Log::Info("history is too large, clean all %s", dir.c_str());
  }
}

int TaskSchedule::Fetch(bool from_kick) {
  int task_size = 0;

  if (from_kick)
    task_size = FetchTasks("kickoff");
  else
    task_size = FetchTasks("period");

  for (int i =  0; i < 1 &&  from_kick  && task_size == 0 ;i++ ) {
    std::this_thread::sleep_for(std::chrono::seconds(3));
    task_size = FetchTasks("kickoff");
  }
  return task_size;
}

void TaskSchedule::Schedule(RunTaskInfo task_info) {

	MutexLocker( &m_mutex ) {
		if ( m_tasklist.find(task_info.task_id) != m_tasklist.end() ) {
			return;
		}
		BaseTask* task = Singleton<TaskFactory>::I().CreateTask(task_info);
		if ( task == nullptr ) {
			return;
		}

		m_tasklist[task_info.task_id] = task;
		if ( task_info.cronat.empty() ) {
			DispatchTask(task);
		}
		else {
			task->timer = (void*)Singleton<TimerManager>::I().createTimer( [this,task] {
				DispatchTask(task);
			}, task_info.cronat );
		}
	}
}

void TaskSchedule::Cancel(StopTaskInfo task_info) {
  Log::Info("stop-task %s", task_info.task_id.c_str());
  BaseTask* task = nullptr;
  MutexLocker( &m_mutex ) {
	  std::map<std::string, BaseTask*>::iterator it;
	  it = m_tasklist.find(task_info.task_id);
	  if ( it == m_tasklist.end() ) {
		  return;
	  }
	  task = it->second;
  }
  task->Cancel();
}


void TaskSchedule::Execute( BaseTask* task ) {
	if ( !task->canceled ) {
		task->Run();
		MutexLocker(&m_mutex) {
			if ( !task->timer ) { //非周期任务,执行完删除
				m_tasklist.erase(task->task_info.task_id);
				Singleton<TaskFactory>::I().DeleteTask(task);
			}
		};
		return;
	}
	//任务被取消
	MutexLocker( &m_mutex ) {
    Log::Info("stop-task %s", task->task_info.task_id.c_str());
		if (task->timer) {
			Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)task->timer);
		}
		m_tasklist.erase(task->task_info.task_id);
		Singleton<TaskFactory>::I().DeleteTask(task);
	};		
};

void TaskSchedule::DispatchTask(BaseTask* task) {
#if defined(_WIN32)

	std::thread ([this,task] {
		Execute(task);
	}).detach();

#else
	struct Args {
		TaskSchedule* pthis;
		BaseTask*     task;
	};

	Args* args  = new Args();
	args->pthis = this;
	args->task  = task;

	pthread_t  tid;
	pthread_create(&tid, nullptr, [](void* args)->void* {
		((Args*)args)->pthis->Execute(((Args*)args)->task);
		delete(Args*)args;
	}, args);
	pthread_detach(tid);

#endif
};

int TaskSchedule::FetchTasks(std::string reason) {
  std::vector<StopTaskInfo> stop_tasks;
  std::vector<RunTaskInfo> run_tasks;
  std::vector<SendFile> sendfile_tasks;
  task_engine::TaskFetch task_fetch;

  task_fetch.FetchTaskList(stop_tasks, run_tasks, sendfile_tasks, reason);

  for (size_t i = 0; i < run_tasks.size(); i++) {
    Schedule(run_tasks[i]);
  }

  for (size_t i = 0; i < stop_tasks.size(); i++) {
    Cancel(stop_tasks[i]);
  }

  for (size_t i = 0; i < sendfile_tasks.size(); i++) {
	  doSendFile(sendfile_tasks[i]);
  }
  return run_tasks.size() + stop_tasks.size() + sendfile_tasks.size();
}

}  // namespace task_engine
