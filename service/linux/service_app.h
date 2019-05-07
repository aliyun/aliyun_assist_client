#pragma once
#include <string>
#include <signal.h>
#include "utils/singleton.h"
class ServiceApp {
  friend Singleton<ServiceApp>;
public:
  int   becomeDeamon();
  void  runService();
  void  runCommon();
private:
  void  start();
  static void reopen_fd_to_null(int fd);
  void  onCommand(std::string msg);
  void  onUpdate();
  void  onStop();
  void  doFetchTasks(bool fromKick);
  void  doUpdate();
  
  void  doShutdown();
  void  doReboot();

private:
  void*   m_updateTimer;
  void*   m_fetchTimer;
  void*   m_updateTimeoutTimer;
  void*   m_notifer;
  bool    m_updateFinish;
};
