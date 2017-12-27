// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./timeout_listener.h"

#include <stdio.h>
#include <set>
#include <vector>
#include <string>
#include <functional>
#include <thread>

#include "utils/MutexLocker.h"

#if !defined(_WIN32)
#include <sys/time.h>
#include <unistd.h>
#else
#include "windows.h"
#endif

namespace task_engine {
TimeoutListener::TimeoutListener() {
  m_worker = nullptr;
  m_stop = true;
#if !defined(_WIN32)
  pthread_mutex_init(&mutex, NULL);
  pthread_cond_init(&cond, NULL);
#endif
}

bool  TimeoutListener::Start() {
  m_worker = new std::thread([this]() {
    while (m_stop) {
      WaitCondition();
      NotifyTimer();
#if defined(_WIN32)
      Sleep(1000);
#else
      sleep(1);
#endif
    }
  });
  return true;
}

void  TimeoutListener::Stop() {
  m_stop = true;
  m_worker->join();
  delete m_worker;

  while (!m_queue.empty()) {
    delete m_queue.top();
    m_queue.pop();
  }
}

void* TimeoutListener::CreateTimer(TimeoutNotifier notifier,
    void* ctx, int timeout) {
  const char*  err = nullptr;

  time_t   now = time(0);
  time_t next = now + timeout;

  TimeoutObject* obj = new TimeoutObject();
  obj->time = next;
  obj->context = ctx;
  obj->notifier = notifier;


  AutoMutexLocker(&m_mutex) {
    m_queue.push(obj);
  }
#if defined(_WIN32)
  m_cv.notify_one();
#else
  pthread_cond_signal(&cond);
#endif
  return obj;
}

void TimeoutListener::NotifyTimer() {
  std::vector<TimeoutObject*> execute_list;
  AutoMutexLocker(&m_mutex) {
    time_t  now = time(0);
    while (!m_queue.empty() && now > m_queue.top()->time) {
      execute_list.push_back(m_queue.top());
      m_queue.pop();
    }
  }

  for (int i = 0; i != execute_list.size(); i++) {
    execute_list[i]->notifier(execute_list[i]->context);
  }
}

void TimeoutListener::WaitCondition() {
  time_t   minus = 60 * 60;
  time_t   now = time(0);
  AutoMutexLocker(&m_mutex) {
    if (!m_queue.empty()) {
      minus = m_queue.top()->time - now;
    }
  }

  if(minus <= 0) {
    return;
  };
#if defined(_WIN32)
  std::mutex dumy;
  std::unique_lock<std::mutex> lock(dumy);
  m_cv.wait_for(lock, std::chrono::seconds(minus));
#else
  pthread_mutex_lock(&mutex);

  struct timespec   ts;
  struct timeval    tp;

  gettimeofday(&tp, NULL);
  ts.tv_sec = tp.tv_sec;
  ts.tv_nsec = tp.tv_usec * 1000;
  ts.tv_sec += minus;

  pthread_cond_timedwait(&cond, &mutex, &ts);
  pthread_mutex_unlock(&mutex);
#endif
  return;
}

}  // namespace task_engine
