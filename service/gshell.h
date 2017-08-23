// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_SERVICE_GSHELL_H_
#define CLIENT_SERVICE_GSHELL_H_
#include <string>
#include <functional>

#include "json11/json11.h"

#ifdef _WIN32
#include <windows.h>
#endif

using std::string;
#define THREAD_SLEEP_TIME 500

struct Error {
  Error() {
    m_Class = "GenericError";
    m_Decs = "";
  }

  void  SetClass(string Class) {
    m_Class = Class;
  }

  void   SetDesc(string Desc) {
    m_Decs = Desc;
  }

  json11::Json  Json() {
    return json11::Json::object{ {
              "error",
              json11::Json::object {
                { "class", m_Class },
                { "desc",  m_Decs }
            }
        } };
  }
 private:
  string  m_Class;
  string  m_Decs;
};

typedef std::function<void(void)> KICKER;

class Gshell {
 public:
  explicit Gshell(KICKER kicker);
  ~Gshell();
  bool  Poll();

 private:
  void  Parse(string input, string& output);
  void  QmpGuestCommand(json11::Json  arguments, string& output);
  void  QmpGuestShutdown(json11::Json arguments, string& output);
  void  QmpGuestSync(json11::Json  arguments,string& output);

#ifdef _WIN32
bool  EnablePrivilege(const char *name, Error& errp);
#else
  void reopen_fd_to_null(int fd);
  bool ga_wait_child(pid_t pid, int *status);
#endif

 private:

#ifdef _WIN32
  HANDLE   m_hFile;
#else
  int      m_hFile;
#endif

  KICKER m_kicker;
};

#endif  // CLIENT_SERVICE_GSHELL_H_
