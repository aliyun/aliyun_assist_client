#ifndef _singleton_h_
#define _singleton_h_
#include <mutex>

template<class T>
class Singleton {
 public:
  static T& I() {
    m_lock.lock();
    if ( !m_object ) m_object = new T;
    m_lock.unlock();
    return *m_object;
  };
 private:
  static std::mutex  m_lock;
  static T*          m_object;
};

template<class T>
T* Singleton<T>::m_object = nullptr;

template<class T>
std::mutex Singleton<T>::m_lock;

#endif


