#ifndef SERVICE_BASE_H_
#define SERVICE_BASE_H_

#include <windows.h>
#include <tchar.h>
#include <string>


#ifdef UNICODE
typedef  std::wstring  tstring;
#else
typedef  std::string   tstring;
#endif


class ServiceBase {
 public:
  ServiceBase(const ServiceBase& other) = delete;
  ServiceBase& operator=(const ServiceBase& other) = delete;

  ServiceBase(ServiceBase&& other) = delete;
  ServiceBase& operator=(ServiceBase&& other) = delete;

  virtual ~ServiceBase() {}
  
  bool run() {
    return run(this);
  }

  const tstring& GetName() const { return m_name; }

 
 protected:
  ServiceBase(const tstring& name,
              DWORD dwErrCtrlType = SERVICE_ERROR_NORMAL,
              DWORD dwAcceptedCmds = SERVICE_ACCEPT_STOP);

  void setStatus(DWORD dwState, DWORD dwErrCode = NO_ERROR, DWORD dwWait = 0);
  void ServiceBase::writeToEventLog(const tstring& msg, WORD type);
 
  virtual void onStart(DWORD argc, TCHAR* argv[]) = 0;
  virtual void onStop() {}
  virtual void onPause() {}
  virtual void onContinue() {}
  virtual void onShutdown() {}

  virtual void onSessionChange(DWORD /*evtType*/,
                               WTSSESSION_NOTIFICATION* /*notification*/) {}
 private:
  static void  WINAPI svcMain(DWORD argc, TCHAR* argv[]);
  static DWORD WINAPI serviceCtrlHandler(DWORD ctrlCode, DWORD evtType,
                                         void* evtData, void* context);

  static bool run(ServiceBase* svc);

  void start(DWORD argc, TCHAR* argv[]);
  void stop();

  tstring m_name;
  DWORD   m_dwErrorCtrlType;


  bool m_hasDepends = false;
  bool m_hasAcc = false;
  bool m_hasPass = false;

  SERVICE_STATUS		m_svcStatus;
  SERVICE_STATUS_HANDLE m_svcStatusHandle;
  static ServiceBase*	m_service;
};

#endif // SERVICE_BASE_H_
