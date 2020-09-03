// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_SERVICE_GSHELL_H_
#define CLIENT_SERVICE_GSHELL_H_
#include <string>
#include <functional>
#include <windows.h>
#include <thread>

#include "../task_notifer.h"
#include "json11/json11.h"
#include <mutex>
#include "utils/MutexLocker.h"

using std::string;
using std::thread;
#define THREAD_SLEEP_TIME 100
class KvmNotifer :public TaskNotifer {

 public:
	KvmNotifer();

	bool init(function<void(const char*)> callback);
	void unit();
  bool is_stopped();
  void set_stop();

 private:
  bool  poll();
  void  parse(string input, string& output);
  void  onGuestCommand(json11::Json  arguments, string& output);
  void  onGuestShutdown(json11::Json arguments, string& output);
  void  onGuestSync(json11::Json  arguments,string& output);
  bool  enablePrivilege(const char *name);

 private:
  HANDLE	m_hFile;
  bool      m_stop;
  thread*   m_worker;
  std::mutex m_mutex;
  function<void(const char*)>    m_callback;
};
#endif  // CLIENT_SERVICE_GSHELL_H_
