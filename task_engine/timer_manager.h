// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_TASK_ENGINE_PLUGIN_TIMER_MANAGER_H_
#define CLIENT_TASK_ENGINE_PLUGIN_TIMER_MANAGER_H_
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

struct  Timer;
typedef std::function<void()> Callback;

class TimerManager {
  friend Singleton<TimerManager>;
 public:
  bool    start();
  void    stop();
  Timer*  createTimer(Callback callback, std::string cronat);
  Timer*  createTimer(Callback callback, int interval);
  void    deleteTimer(Timer* timer);

 private:
  void            updateTime(Timer* timer);
  void            checkTimer();
  void            wait();
  void            notifty();

  static void*    worker(void* args);

 private:
  TimerManager();
  std::mutex			m_mutex;
  bool					m_stop;
  std::vector<Timer*>   m_queue;
#if defined(_WIN32)
  std::condition_variable  m_cv;
#else
  pthread_cond_t      cond;
  pthread_mutex_t     mutex;
#endif
  std::vector<Timer*>  m_deleteList;
};

}  // namespace task_engine
#endif  // CLIENT_TASK_ENGINE_PLUGIN_TIMER_MANAGER_H_
