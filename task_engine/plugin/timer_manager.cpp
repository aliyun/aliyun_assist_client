// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./timer_manager.h"

#include <stdio.h>
#include <set>
#include <vector>
#include <string>
#include <functional>
#include <thread>

#include "utils/MutexLocker.h"
#include "ccronexpr/ccronexpr.h"

#if !defined(_WIN32)
#include <sys/time.h>
#include <unistd.h>
#else
#include "windows.h"
#endif

namespace task_engine {
TimerManager::TimerManager() {
  m_worker = nullptr;
  m_stop = true;
#if !defined(_WIN32)
  pthread_mutex_init(&mutex, NULL);
  pthread_cond_init(&cond, NULL);
#endif
}

bool  TimerManager::Start() {
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

void  TimerManager::Stop() {
  m_stop = true;
  m_worker->join();
  delete m_worker;

  while (!m_queue.empty()) {
    delete m_queue.top();
    m_queue.pop();
  }
}

void  TimerManager::DeleteTimer(void* item) {
  AutoMutexLocker(&m_mutex) {
    m_deleted.insert(reinterpret_cast<TimerObject*>(item));
  }
}

void* TimerManager::CreateTimer(TimerNotifier notifier,
    void* ctx, std::string cronat) {
  const char*  err = nullptr;

  time_t   now = time(0);
  time_t next = GetNextTime(const_cast<char*>(cronat.c_str()), &err);
  if (err) {
    return nullptr;
  }

  TimerObject* obj = new TimerObject();
  obj->time = next;
  obj->context = ctx;
  obj->notifier = notifier;
  obj->cronat = cronat;

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

void   TimerManager::NotifyTimer() {
  std::vector<TimerObject*> execute_list;
  AutoMutexLocker(&m_mutex) {
    time_t  now = time(0);
    while (!m_queue.empty() && now > m_queue.top()->time) {
      std::set<TimerObject*>::iterator it = m_deleted.find(m_queue.top());
      if (it == m_deleted.end()) {
        execute_list.push_back(m_queue.top());
      } else {
        m_deleted.erase(it);
        delete m_queue.top();
      }
      m_queue.pop();
    }
  }

  for (int i = 0; i != execute_list.size(); i++) {
    execute_list[i]->notifier(execute_list[i]->context);
    const char* err;
    time_t next = GetNextTime((char*)(execute_list[i]->cronat.c_str()), &err);
    if (err) {
      delete execute_list[i];
      continue;
    }

    AutoMutexLocker(&m_mutex) {
      execute_list[i]->time = next;
      m_queue.push(execute_list[i]);
    }
  }
}

void TimerManager::WaitCondition() {
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


int TimerManager::TimeOffset() {
  time_t gmt, rawtime = time(NULL);
  struct tm *ptm;

#if !defined(WIN32)
  struct tm gbuf;
  ptm = gmtime_r(&rawtime, &gbuf);
#else
  ptm = gmtime(&rawtime);
#endif
  // Request that mktime() looksup dst in timezone database
  ptm->tm_isdst = -1;
  gmt = mktime(ptm);

  return (int)difftime(rawtime, gmt);
}

time_t TimerManager::GetNextTime(char* pattern, const char** err) {
  *err = reinterpret_cast<char*>(0);
  cron_expr* parsed = cron_parse_expr(pattern, err);
  if (*err) {
    return -1;
  }

  int offset = TimeOffset();
  time_t now = time(0) + offset;
  time_t datenext = cron_next(parsed, now);
  return datenext - offset;
}
}  // namespace task_engine
