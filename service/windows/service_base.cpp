#include "service_base.h"
#include <cassert>

ServiceBase* ServiceBase::m_service = nullptr;

ServiceBase::ServiceBase(const tstring& name,
                         DWORD dwErrCtrlType,
                         DWORD dwAcceptedCmds):
   m_name(name),
   m_svcStatusHandle(nullptr) {  

  m_svcStatus.dwControlsAccepted = dwAcceptedCmds;
  m_svcStatus.dwServiceType = SERVICE_WIN32_OWN_PROCESS;
  m_svcStatus.dwCurrentState = SERVICE_START_PENDING;
  m_svcStatus.dwWin32ExitCode = NO_ERROR;
  m_svcStatus.dwServiceSpecificExitCode = 0;
  m_svcStatus.dwCheckPoint = 0;
  m_svcStatus.dwWaitHint = 0;
}

void ServiceBase::setStatus(DWORD dwState, DWORD dwErrCode, DWORD dwWait) {
  m_svcStatus.dwCurrentState = dwState;
  m_svcStatus.dwWin32ExitCode = dwErrCode;
  m_svcStatus.dwWaitHint = dwWait;

  ::SetServiceStatus(m_svcStatusHandle, &m_svcStatus);
}

void ServiceBase::writeToEventLog(const tstring& msg, WORD type) {
  HANDLE hSource = RegisterEventSource(nullptr, m_name.c_str());
  if (hSource) {
    const TCHAR* msgData[2] = { m_name.c_str(), msg.c_str()};
    ReportEvent(hSource, type, 0, 0, nullptr, 2, 0, msgData, nullptr);
    DeregisterEventSource(hSource);
  }
}

void WINAPI ServiceBase::svcMain(DWORD argc, TCHAR* argv[]) {
  assert(m_service);

  m_service->m_svcStatusHandle = ::RegisterServiceCtrlHandlerEx(m_service->GetName().c_str(),
                                                                serviceCtrlHandler, NULL);
  if (!m_service->m_svcStatusHandle) {
    m_service->writeToEventLog(_T("Can't set service control handler"),
                               EVENTLOG_ERROR_TYPE);
    return;
  }
  m_service->start(argc, argv);
}

DWORD WINAPI ServiceBase::serviceCtrlHandler(DWORD ctrlCode, DWORD evtType,
                                             void* evtData, void* context) {
  switch (ctrlCode) {
    case SERVICE_CONTROL_STOP:
      m_service->stop();
    break;
    default:
    break;
  }

  return 0;
}

bool ServiceBase::run(ServiceBase* svc) {
  m_service = svc;
  LPTSTR svcName = (LPTSTR)m_service->GetName().c_str();
  SERVICE_TABLE_ENTRY tableEntry[] = {
    {svcName, svcMain},
    {nullptr, nullptr}
  };
  return ::StartServiceCtrlDispatcher(tableEntry) == TRUE;
}

void ServiceBase::start(DWORD argc, TCHAR* argv[]) {
  setStatus(SERVICE_START_PENDING);
  onStart(argc, argv);
  setStatus(SERVICE_RUNNING);
}

void ServiceBase::stop() {
  setStatus(SERVICE_STOP_PENDING);
  onStop();
  setStatus(SERVICE_STOPPED);
}

