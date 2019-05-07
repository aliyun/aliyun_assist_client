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
#if !defined(_WIN32)
#include <pthread.h>
#include <unistd.h>
#else
#include <windows.h>
#endif

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

void TaskSchedule::FetchPeriodTask() {
  std::vector<TaskInfo> tasks;
  task_engine::TaskFetch task_fetch;
  //task_fetch.FetchPeriodTasks(tasks);
  for (size_t i = 0; i < tasks.size(); i++) {
    Schedule(tasks[i]);
  }
}

int TaskSchedule::Fetch(bool from_kick) {

  std::vector<TaskInfo>  tasks;
  std::vector<TaskInfo>  canceled_tasks;
  task_engine::TaskFetch task_fetch;
  
  task_fetch.FetchTasks(tasks);
  task_fetch.FetchCancledTasks(canceled_tasks);

  for ( size_t i = 0; i < tasks.size(); i++ ) {
    Schedule(tasks[i]);
  }

  for (size_t i = 0; i < canceled_tasks.size(); i++) {
    Cancel(canceled_tasks[i]);
  }
  int task_size = tasks.size() + canceled_tasks.size();

  for (int i =  0; i < 3 &&  from_kick  && task_size == 0 ;i++ ) {
	  std::this_thread::sleep_for(std::chrono::seconds(3));
	  task_size = Fetch(false);
  }
  return task_size;
}

void TaskSchedule::Schedule(TaskInfo task_info) {

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

void TaskSchedule::Cancel(TaskInfo task_info) {
  Log::Error("cancel task taskid:%s", task_info.task_id.c_str());
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


}  // namespace task_engine
