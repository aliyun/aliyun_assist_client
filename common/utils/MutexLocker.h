#pragma once;

#include <mutex>

struct MutexLocker {
  MutexLocker(std::mutex* mutex) :m_mark(true), m_mutex(mutex) {
    m_mutex->lock();
  };

  ~MutexLocker() {
    m_mutex->unlock();
  };

  std::mutex* m_mutex;
  bool        m_mark;
};

#define AutoMutexLocker(mutex) for(MutexLocker locker(mutex); locker.m_mark; locker.m_mark = false)





