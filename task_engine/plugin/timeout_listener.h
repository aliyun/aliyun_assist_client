// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_TIMEOUT_LISTENER_H_
#define CLIENT_TASK_ENGINE_PLUGIN_TIMEOUT_LISTENER_H_
#include <queue>
#include <set>
#include <vector>
#include <mutex>
#include <thread>
#include <string>
#include <condition_variable>
#include <functional>
#include "utils/singleton.h"

#if !defined(_WIN32)
#include <pthread.h>
#endif

namespace task_engine {

struct TimeoutObject;
struct TimeoutComprator;

typedef std::priority_queue<TimeoutObject*, std::vector<TimeoutObject*>, TimeoutComprator> TimeoutQueue;
typedef std::function<void(void*)> TimeoutNotifier;

struct TimeoutObject {
  time_t         time;
  TimeoutNotifier  notifier;
  void*          context;
};

struct TimeoutComprator {
  bool operator() (TimeoutObject* a, TimeoutObject* b) {
    return a->time > b->time;
  }
};

class TimeoutListener {
  friend Singleton<TimeoutListener>;
 public:
  void*   CreateTimer(TimeoutNotifier notifier, void* ctx, int timeout);
  void    DeleteTimer(void* timer);
  bool    Start();
  void    Stop();

 private:
  void            WaitCondition();
  void            NotifyTimer();

 private:
  TimeoutListener();
  std::mutex          m_mutex;
  bool                m_stop;
  TimeoutQueue          m_queue;
#if defined(_WIN32)
  std::condition_variable  m_cv;
#else
  pthread_cond_t cond;
  pthread_mutex_t     mutex;
#endif
  std::thread*             m_worker;
};

}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_TIMEOUT_LISTENER_H_
