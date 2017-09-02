// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_TIMER_MANAGER_H_
#define CLIENT_TASK_ENGINE_PLUGIN_TIMER_MANAGER_H_
#include <queue>
#include <set>
#include <vector>
#include <mutex>
#include <chrono>
#include <thread>
#include <string>
#include <condition_variable>
#include <functional>
#include "utils/singleton.h"

#if !defined(_WIN32)
#include <pthread.h>
#endif

namespace task_engine {

struct TimerObject;
struct TimerComprator;

typedef std::priority_queue<TimerObject*, std::vector<TimerObject*>, TimerComprator> TimerQueue;
typedef std::function<void(void*)> TimerNotifier;

struct TimerObject {
  time_t         time;
  TimerNotifier  notifier;
  void*          context;
  std::string    cronat;
};

struct TimerComprator {
  bool operator() (TimerObject* a, TimerObject* b) {
    return a->time > b->time;
  }
};

class TimerManager {
  friend Singleton<TimerManager>;
 public:
  void*   CreateTimer(TimerNotifier notifier, void* ctx, std::string cronat);
  void    DeleteTimer(void* timer);
  bool    Start();
  void    Stop();

 private:
  void            WaitCondition();
  void            NotifyTimer();
  time_t          GetNextTime(char* pattern, const char** err);
  int             TimeOffset();

 private:
  TimerManager();
  std::mutex          m_mutex;
  bool                m_stop;
  TimerQueue          m_queue;
#if defined(_WIN32)
  std::condition_variable  m_cv;
#else
  pthread_cond_t cond;
  pthread_mutex_t     mutex;
#endif
  std::set<TimerObject*>   m_deleted;
  std::thread*             m_worker;
};

}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_TIMER_MANAGER_H_
