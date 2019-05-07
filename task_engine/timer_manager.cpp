// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./timer_manager.h"

#include <stdio.h>
#include <vector>
#include <string>
#include <functional>
#include <thread>
#include <algorithm>

#include "utils/MutexLocker.h"
#include "ccronexpr/ccronexpr.h"

#if !defined(_WIN32)
#include <sys/time.h>
#include <unistd.h>
#else
#include "windows.h"
#endif

namespace task_engine {

	
struct Timer {
	time_t         time;
	Callback	   notifier;
	void*          context;
	cron_expr*	   expr;
	int            interval;
};

bool comprator(Timer* a, Timer* b) {
	return a->time > b->time;
}



TimerManager::TimerManager() {
  m_stop = false;
#if !defined(_WIN32)
  pthread_mutex_init(&mutex, NULL);
  pthread_cond_init(&cond, NULL);
#endif
}

	
bool  TimerManager::start() {
	
#ifdef _WIN32
	std::thread(worker, (void*) this).detach();	
#else
	pthread_t tid;
	pthread_create(&tid, nullptr, worker, (void*) this);
	pthread_detach(tid);
#endif
  return true;
}

void  TimerManager::stop() {
  m_stop = true;
  MutexLocker( &m_mutex ) {
	  std::vector<Timer*>::iterator it;
	  for (it = m_queue.begin(); it != m_queue.end(); ) {
		  delete *it;
		  it = m_queue.erase(it);
	  }
  }
}
	
void*  TimerManager::worker(void* args) {
	TimerManager* pthis = (TimerManager*) args;
	while ( !pthis->m_stop ) {
		pthis->wait();
		pthis->checkTimer();
	}	
	return NULL;
};

void  TimerManager::deleteTimer(Timer* timer) {
	bool   bfind = false;
	MutexLocker( &m_mutex ) {
	  std::vector<Timer*>::iterator it = std::find(m_queue.begin(), m_queue.end(), timer);
	  if ( it != m_queue.end() ){
		   m_queue.erase(it);
		   delete timer;
	  }
  }
  notifty();
}

Timer* TimerManager::createTimer(Callback notifier, std::string cronat) {
  const char*  err = nullptr;
  cron_expr*  expr = cron_parse_expr(cronat.c_str(), &err);
  if ( err ) {
	 return nullptr;
  }
	
  Timer* timer    = new Timer();
  timer->expr     = expr;
  timer->time     = cron_next(expr, time(0));
  timer->notifier = notifier;
  timer->interval = 0;

  MutexLocker(&m_mutex) {
	  m_queue.insert(
		  std::upper_bound(m_queue.begin(), m_queue.end(), timer, comprator),
		  timer);
  }

  notifty();
  return timer;
}


Timer * TimerManager::createTimer(Callback callback, int interval) {
	if ( interval <=0 ) {
		return nullptr;
	}
	time_t   next   = time(0) + interval;
	Timer* timer    = new Timer();
	timer->time     = next;
	timer->notifier = callback;
	timer->expr     = nullptr;
	timer->interval = interval;
	
	MutexLocker(&m_mutex) {
		m_queue.insert(
			std::upper_bound(m_queue.begin(), m_queue.end(), timer, comprator), 
			timer);
	}
	notifty();
	return timer;
};




void  TimerManager::updateTime(Timer* timer) {
	if ( timer->interval ) {
		timer->time = timer->interval + time(0);
	}
	else {
		timer->time = cron_next(timer->expr, time(0));
	}
};

void  TimerManager::checkTimer() {

  time_t now = time(0);
  std::vector<Timer*> notifyList;

  MutexLocker( &m_mutex ) {
 
    while ( !m_queue.empty() && now > m_queue.back()->time ) {
		notifyList.push_back( m_queue.back() );
		m_queue.pop_back();
    }

	for ( int i = 0; i != notifyList.size(); i++ ) {
		Timer* timer = notifyList[i];
		timer->notifier();
		updateTime(timer);
		m_queue.insert(
			std::upper_bound(m_queue.begin(), m_queue.end(), timer, comprator),
			timer);
	}
  }
}

void TimerManager::wait() {
  
  time_t   minus = 60 * 60;
  time_t   now   = time(0);

  MutexLocker(&m_mutex) {
    if ( !m_queue.empty() ) {
       minus = m_queue.back()->time - now;
    }
  }

  if( minus <= 0 ) {
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
  ts.tv_sec  = tp.tv_sec;
  ts.tv_nsec = tp.tv_usec * 1000;
  ts.tv_sec += minus;
  pthread_cond_timedwait(&cond, &mutex, &ts);
  pthread_mutex_unlock(&mutex);
#endif
    return;
}

void  TimerManager::notifty() {
#if defined(_WIN32)
	m_cv.notify_one();
#else
	pthread_cond_signal(&cond);
#endif
};
	
}  // namespace task_engine
