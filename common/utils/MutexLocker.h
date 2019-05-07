#ifndef _mutex_locker_
#define _mutex_locker_
				  
#include <mutex>

struct MutexLockerHelper {
  MutexLockerHelper(std::mutex* mutex) :m_mark(true), m_mutex(mutex) {
    m_mutex->lock();
  };

  ~MutexLockerHelper() {
    m_mutex->unlock();
  };

  std::mutex* m_mutex;
  bool        m_mark;
};

#define MutexLocker(mutex) for(MutexLockerHelper locker(mutex); locker.m_mark; locker.m_mark = false)
#endif // 




