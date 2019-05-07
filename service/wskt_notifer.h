// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef WSKT_TASKNOTIFIER_H_
#define WSKT_TASKNOTIFIER_H_
#include <string>
#include <functional>
#include <thread>

#include "json11/json11.h"
#include "task_notifer.h"


class WsktNotifer :public TaskNotifer {

 public:
	 WsktNotifer();

	bool init(function<void(const char*)> callback);
	void unit();

 private:

	void  poll();  
	void  handle_message(const std::string & message);

 private:
  function<void(char*)>    m_callback;
  bool                m_stop;
  char*               m_path;
  std::thread*        m_worker;
};
#endif  // CLIENT_SERVICE_GSHELL_H_
