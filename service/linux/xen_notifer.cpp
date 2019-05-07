// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "xen_notifer.h"
#include <string>
#include <thread>
#include "utils/Log.h"

#define XS_PATH_CMDSTATEIN   "control/shell/statein"
#define XS_PATH_CMDSTATEOUT  "control/shell/stateout"
#define XS_PATH_CMDSTDIN     "control/shell/stdin"
#define XS_PATH_CMDSTDOUT    "control/shell/stdout"
#define XS_PATH_SHUTDOWN     "control/shutdown"



#define LENGTH_TIMESTAMP 15
#define CMD_MAX_LENGTH   850
#define BUFFER_SIZE      850
#define STATE_ENABLE     "1"

/*Error message*/
#define ERR_CMD_NOT_SUPPORT     "command is not supported\n"
#define SUC_KICK_VM             "\"result\":8, execute kick_vm success\n"

XenNotifer::XenNotifer() {
  m_stop   = false;
}

bool XenNotifer::init(function<void(const char*)> callback) {
  m_eventWorker = new std::thread([this]() {
    pool_shell();
  });

  m_checkWorker = new std::thread([this]() {
    pool_state();
  });
	
  m_callback = callback;
  return true;
}

void XenNotifer::unit() {
  m_stop = true;

  if (m_eventWorker) {
    m_eventWorker->join();
  }
}

void  XenNotifer::pool_shell() {
  struct xs_handle *watch_xsh;
  struct xs_handle *xsh;
  char **res;
  const char* token = "0";
  unsigned int num;
  char* buf;
  unsigned int len;

  Log::Info("pool_shell start");
  if ((watch_xsh = xs_domain_open()) == NULL) {
    Log::Error("xs_domain_open failed: %s", strerror(errno));
    return;
  }

  xs_watch(watch_xsh, XS_PATH_CMDSTDIN, token);

  if ((xsh = xs_domain_open()) == NULL) {
    Log::Error("Connect to xenbus failed: %s", strerror(errno));
    return;
  }

  while (!m_stop) {
    if ((res = xs_read_watch(watch_xsh, &num)) == NULL)
      continue;

    buf = (char*)xs_read(xsh, XBT_NULL, XS_PATH_CMDSTDIN, &len);

    if (buf == NULL) {
      free(res);
      continue;
    }

    Log::Info("receive event: %s", buf);
    if (strstr(buf, "kick_vm")) {
      m_callback("kick_vm");
      write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTDOUT, SUC_KICK_VM,
          strlen(SUC_KICK_VM), buf);
    } else {
      write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTDOUT, ERR_CMD_NOT_SUPPORT,
          strlen(ERR_CMD_NOT_SUPPORT), buf);
    }

    if (buf != NULL)
      free(buf);
    free(res);
  }

  Log::Info("pool_shell end");
  return;
}

void  XenNotifer::pool_state() {
  struct xs_handle *watch_xsh;
  struct xs_handle *xsh;
  char **res;
  const char* token = "0";
  unsigned int num;

  Log::Info("pool_state start");
  if ((watch_xsh = xs_domain_open()) == NULL) {
    Log::Error("xs_domain_open failed: %s", strerror(errno));
    return;
  }
  xs_watch(watch_xsh, XS_PATH_CMDSTATEIN, token);

  if ((xsh = xs_domain_open()) == NULL) {
    Log::Error("Connect to xenbus failed: %s", strerror(errno));
    return;
  }

  write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTATEOUT, STATE_ENABLE,
      strlen(STATE_ENABLE), NULL);

  while (!m_stop) {
    if ((res = xs_read_watch(watch_xsh, &num)) == NULL)
      continue;

    write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTATEOUT, STATE_ENABLE,
        strlen(STATE_ENABLE), NULL);

    free(res);
  }

  Log::Info("pool_state end");
  return;
}


bool XenNotifer::write_xenstore(struct xs_handle *h,
    xs_transaction_t t,
    const char *path,
    const void *data,
    unsigned int len,
    const char *ptimestamp) {
  char writebuf[BUFFER_SIZE + LENGTH_TIMESTAMP];
  int str_len;

  if (ptimestamp != NULL) {
    if (strlen(ptimestamp) >= LENGTH_TIMESTAMP)
      memcpy(writebuf, ptimestamp, LENGTH_TIMESTAMP);
    memcpy(writebuf + LENGTH_TIMESTAMP, data, len);
    str_len = len + LENGTH_TIMESTAMP;
  }
  else {
    memcpy(writebuf, data, len);
    str_len = len;
  }

  Log::Info("xs_write: [%s] [%.*s] [%d]", path, str_len, writebuf, str_len);
  return xs_write(h, t, path, writebuf, str_len);
}

