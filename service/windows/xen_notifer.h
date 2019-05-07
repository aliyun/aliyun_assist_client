// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef XEN_TASKNOTIFIER_H_
#define XEN_TASKNOTIFIER_H_
#include <string>
#include <functional>
#include <thread>

#include "json11/json11.h"

#include <windows.h>
#include "../task_notifer.h"





using std::string;

#define THREAD_SLEEP_TIME 100
class XenNotifer :public TaskNotifer {

 public:
	XenNotifer();

	bool init(function<void(const char*)> callback);
	void unit();

 private:
	char* xb_read(HANDLE handle, char *path);
	int   xb_write(HANDLE handle, char *path, char* info, size_t infoLen);
	int   xb_wait_event(HANDLE handle);
	int   xb_add_watch(HANDLE handle, char *path);
	char* get_xen_interface_path();
	void  write_xenstore(HANDLE handle, char* path, char* buf, size_t bufLen, char* ptimeStamp);
	void  pool_shell();
	void  pool_shutdown();
	void  pool_state();

 private:
  bool            m_stop;
  char*           m_path;

  std::thread*    m_checkWorker;
  std::thread*    m_eventWorker;
  std::thread*    m_shutdownWorker;
  function<void(const char*)>    m_callback;
};
#endif  // CLIENT_SERVICE_GSHELL_H_
