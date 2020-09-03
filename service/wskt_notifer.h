// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef WSKT_TASKNOTIFIER_H_
#define WSKT_TASKNOTIFIER_H_
#include <string>
#include <functional>
#include <thread>
#include <mutex>

#include "json11/json11.h"
#include "task_notifer.h"
#if !defined(_WIN32)
#include <pthread.h>
#endif


class WsktNotifer :public TaskNotifer {

 public:
	 WsktNotifer();

	bool init(function<void(const char*)> callback);
	void unit();
  bool is_stopped();
  void set_stop();

 private:

  static void* poll(void* args);
	void  handle_message(const std::string & message);

 private:
  function<void(char*)>    m_callback;
  bool                m_stop;
  char*               m_path;
  std::mutex m_mutex;
#if defined(_WIN32)
  std::thread*        m_worker;
#else
  pthread_t m_worker;
#endif
};
#endif  // CLIENT_SERVICE_GSHELL_H_
