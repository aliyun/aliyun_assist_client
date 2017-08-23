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

namespace task_engine {
TimerManager::TimerManager() {
  m_stop = true;
}

bool  TimerManager::Start() {
  m_worker = new std::thread([this]() {
    while (m_stop) {
      WaitCondition();
      NotifyTimer();
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

  m_cv.notify_one();
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

  std::mutex dumy;
  std::unique_lock<std::mutex> lock(dumy);
  m_cv.wait_for(lock, std::chrono::seconds(minus));
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
