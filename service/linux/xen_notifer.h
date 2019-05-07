// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef XEN_TASKNOTIFIER_H_
#define XEN_TASKNOTIFIER_H_
#include <string>
#include <functional>
#include <thread>

#include "json11/json11.h"
#include "../task_notifer.h"
#include "xs/xs.h"

using std::string;

#define THREAD_SLEEP_TIME 100
class XenNotifer :public TaskNotifer {
 public:
  XenNotifer();

  bool init(function<void(const char*)> callback);
  void unit();

 private:
  void  pool_shell();
  void  pool_state();
  void  pool_shutdown();
  bool  write_xenstore(struct xs_handle *h,
      xs_transaction_t t,
      const char *path,
      const void *data,
      unsigned int len,
      const char *ptimestamp);

 private:
  bool            m_stop;

  std::thread*    m_checkWorker;
  std::thread*    m_eventWorker;
  function<void(const char*)>    m_callback;
};
#endif  // CLIENT_SERVICE_GSHELL_H_
