// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <stdio.h>
#include <wchar.h>
#include <DbgHelp.h>
#include <time.h>
#include <powrprof.h>
#include <string>
#include <thread>

#include "jsoncpp/json.h"
#include "utils/CheckNet.h"
#include "utils/FileVersion.h"
#include "utils/http_request.h"
#include "utils/OsVersion.h"
#include "utils/Log.h"
#include "json11/json11.h"
#include "./gshell.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/singleton.h"
#include "./schedule_task.h"
#include "optparse/OptionParser.h"
#include "curl/curl.h"
#include "plugin/timer_manager.h"
#include "utils/dump.h"
#include "utils/Encode.h"
#include "../VersionInfo.h"
#include "./xs_shell.h"
#include "./xs.h"

#define PROCESS_MAX_DURATION 60 * 60 * 1000
#define BUFSIZE MAX_PATH
#define LOGFILE "aliyun_assist_service.log"
#define UPDATER_TIMER_DURATION 60*60*1000
#define UPDATER_TIMER_DUETIME 15*1000
#define ROLL_TIMER_DURATION 30*60*1000
#define ROLL_TIMER_DUETIME 20*1000
#define UPDATERFILE "aliyun_assist_update.exe"
#define UPDATERCOMMANDLINE " --check_update"

#define DEV_VIRTIO
#define DEV_SERIAL "\\\\.\\Global\\org.qemu.guest_agent.0"


using  std::string;

HANDLE gTerminateEvent = NULL;
volatile long gMsgCount = 0;

BOOL pauseService = FALSE;
BOOL runningService = FALSE;
BOOL terminatingService = FALSE;
WCHAR * serviceName = L"aliyun_assist_service";
SERVICE_STATUS serviceStatus;
SERVICE_STATUS_HANDLE serviceStatusHandle;
HANDLE producerThreadHandle = NULL;
HANDLE consumerThreadHandle = NULL;
HANDLE updaterThreadHandle = NULL;
HANDLE serverSyncThreadHandle = NULL;
HANDLE xenThreadHandle = NULL;
HANDLE xenCmdExecThread = NULL;
HANDLE xenCmdReadThread = NULL;
th_param param;

void Terminate(DWORD errCode);
BOOL ServerMsgSyncUp();
VOID CALLBACK  ServerSyncTimerRoutine(PVOID lpParam, BOOLEAN TimerOrWaitFired);


BOOL LaunchProcessAndWaitForExit(CHAR* moduleName, CHAR* commandLines) {
  if ( !moduleName || !commandLines ) {
    return FALSE;
  }

  STARTUPINFOA si;
  PROCESS_INFORMATION pi;

  ZeroMemory(&si, sizeof(si));
  si.cb = sizeof(si);
  ZeroMemory(&pi, sizeof(pi));

  CHAR Buffer[BUFSIZE];
  DWORD dwRet = GetModuleFileNameA(NULL, Buffer, BUFSIZE);

  if (dwRet == 0 || dwRet > BUFSIZE) {
    Log::Error("get module file name failed,error code is %d", GetLastError());
    return FALSE;
  }

  string filePath = Buffer;
  filePath = filePath.substr(0, filePath.find_last_of('\\',
    filePath.length()) + 1);
  filePath = filePath + moduleName;

  string command_line = filePath;
  command_line += commandLines;

  if (!CreateProcessA(nullptr,   // No module name (use command line)
    (LPSTR)command_line.c_str(),        // Command line
    NULL,           // Process handle not inheritable
    NULL,           // Thread handle not inheritable
    FALSE,          // Set handle inheritance to FALSE
    0,              // No creation flags
    NULL,           // Use parent's environment block
    NULL,           // Use parent's starting directory
    &si,            // Pointer to STARTUPINFO structure
    &pi)           // Pointer to PROCESS_INFORMATION structure
    ) {
    Log::Error("createProcess failed,error code is %d", GetLastError());
    return FALSE;
  }

  // Wait until child process exits.
  DWORD ret = WaitForSingleObject(pi.hProcess, PROCESS_MAX_DURATION);

  // Close process and thread handles.
  CloseHandle(pi.hProcess);
  CloseHandle(pi.hThread);

  // If the object is not sigalled, we think the call is failure.
  if (ret != WAIT_OBJECT_0) {
    Log::Warn("process is not completed correctly,error code is %d",
        GetLastError());
    return FALSE;
  }

  return TRUE;
}


DWORD ProducerThreadFunc(LPDWORD param) {
    Gshell gshell([]() {
        InterlockedIncrement(&gMsgCount);
    });

    bool result = true;
    while (!terminatingService && result) {
        result = gshell.Poll();
    }

    return TRUE;
}

DWORD ConsumerThreadFunc(LPDWORD param) {
  while (!terminatingService) {
    if (gMsgCount > 0) {
      Singleton<task_engine::TaskSchedule>::I().Fetch();
      Log::Info("Received kick msg");
      InterlockedDecrement(&gMsgCount);
    }
    Sleep(THREAD_SLEEP_TIME);
  }

  return TRUE;
}

BOOL ServerMsgSyncUp() {
  Singleton<task_engine::TaskSchedule>::I().Fetch();
  Log::Info("Poll to fetch task");

  return TRUE;
}

VOID CALLBACK  ServerSyncTimerRoutine(PVOID lpParam, BOOLEAN TimerOrWaitFired) {
  BOOL ret = FALSE;

  ret = ServerMsgSyncUp();

  if (ret) {
    Log::Info("Aliyun assist sync up with server successfully");
  } else {
    Log::Warn("Aliyun assist failed to sync up with server");
  }
}

DWORD ServerSyncThreadFunc(LPDWORD param) {
  HANDLE hTimer = NULL;
  HANDLE hTimerQueue = NULL;
  DWORD errCode;

  // Create the timer queue.
  hTimerQueue = CreateTimerQueue();
  if (NULL == hTimerQueue) {
    errCode = GetLastError();
    Log::Error("CreateTimerQueue failed, error code is %d", errCode);
    Terminate(errCode);
    return FALSE;
  }

  // Set a timer to call the timer routine in
  // UPDATER_TIMER_DURATION milliseconds.
  if (!CreateTimerQueueTimer(&hTimer, hTimerQueue,
    (WAITORTIMERCALLBACK)ServerSyncTimerRoutine, NULL,
    ROLL_TIMER_DUETIME, ROLL_TIMER_DURATION, 0)) {
    errCode = GetLastError();
    Log::Error("CreateTimerQueueTimer failed, error code is %d", errCode);
    DeleteTimerQueue(hTimerQueue);
    Terminate(errCode);
    return FALSE;
  }


  if (WaitForSingleObject(gTerminateEvent, INFINITE) != WAIT_OBJECT_0) {
    Log::Error("WaitForSingleObject failed, error code is %d", GetLastError());
  }

  if (!DeleteTimerQueue(hTimerQueue)) {
    Log::Error("DeleteTimerQueue failed, error code is %d", GetLastError());
  }

  return TRUE;
}

VOID CALLBACK  UpdaterTimerRoutine(PVOID lpParam, BOOLEAN TimerOrWaitFired) {
  BOOL ret = FALSE;

  ret = LaunchProcessAndWaitForExit(UPDATERFILE, UPDATERCOMMANDLINE);

  if (ret) {
    Log::Info("Aliyun assist updated successfully");
  } else {
    Log::Warn("Aliyun assist failed to update");
  }
}

DWORD UpdaterThreadFunc(LPDWORD param) {
  HANDLE hTimer = NULL;
  HANDLE hTimerQueue = NULL;
  DWORD errCode;

  // Create the timer queue.
  hTimerQueue = CreateTimerQueue();
  if (NULL == hTimerQueue) {
    errCode = GetLastError();
    Log::Error("CreateTimerQueue failed, error code is %d", errCode);
    Terminate(errCode);
    return FALSE;
  }

  // Set a timer to call the timer routine in
  // UPDATER_TIMER_DURATION milliseconds.
  if (!CreateTimerQueueTimer(&hTimer, hTimerQueue,
    (WAITORTIMERCALLBACK)UpdaterTimerRoutine, NULL,
    UPDATER_TIMER_DUETIME, UPDATER_TIMER_DURATION, 0)) {
    errCode = GetLastError();
    Log::Error("CreateTimerQueueTimer failed, error code is %d", errCode);
    DeleteTimerQueue(hTimerQueue);
    Terminate(errCode);
    return FALSE;
  }


  if (WaitForSingleObject(gTerminateEvent, INFINITE) != WAIT_OBJECT_0) {
    Log::Error("WaitForSingleObject failed, error code is %d", GetLastError());
  }

  if (!DeleteTimerQueue(hTimerQueue)) {
    Log::Error("DeleteTimerQueue failed, error code is %d", GetLastError());
  }

  return TRUE;
}

static void
do_hibernate() {
  HANDLE proc_handle = GetCurrentProcess();
  TOKEN_PRIVILEGES *tp;
  HANDLE token_handle;

#ifdef _DEBUG
  Log::Info("proc_handle = %p", proc_handle);
#endif

  if (!OpenProcessToken(proc_handle,
      TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &token_handle)) {
    Log::Error("OpenProcessToken failed, error code is %d", GetLastError());
    return;
  }

#ifdef _DEBUG
  Log::Info("token_handle = %p", token_handle);
#endif

  tp = (TOKEN_PRIVILEGES*)malloc(sizeof(TOKEN_PRIVILEGES) +
      sizeof(LUID_AND_ATTRIBUTES));
  tp->PrivilegeCount = 1;
  if (!LookupPrivilegeValueA(NULL, "SeShutdownPrivilege",
      &tp->Privileges[0].Luid)) {
    Log::Error("LookupPrivilegeValue failed, error code is %d", GetLastError());
    CloseHandle(token_handle);
    return;
  }

  tp->Privileges[0].Attributes = SE_PRIVILEGE_ENABLED;
  if (!AdjustTokenPrivileges(token_handle, FALSE, tp, 0, NULL, NULL)) {
    CloseHandle(token_handle);
    return;
  }

  if (!SetSuspendState(TRUE, FALSE, FALSE)) {
    Log::Error("hibernate failed, error code is %d", GetLastError());
  }

  CloseHandle(token_handle);
}

static void
do_shutdown(BOOL bRebootAfterShutdown) {
  HANDLE proc_handle = GetCurrentProcess();
  TOKEN_PRIVILEGES *tp;
  HANDLE token_handle;

#ifdef _DEBUG
  Log::Info("proc_handle = %p", proc_handle);
#endif

  if (!OpenProcessToken(proc_handle,
      TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &token_handle)) {
    Log::Error("OpenProcessToken failed, error code is %d", GetLastError());
    return;
  }

#ifdef _DEBUG
  Log::Info("token_handle = %p", token_handle);
#endif

  tp = (TOKEN_PRIVILEGES*)malloc(sizeof(TOKEN_PRIVILEGES) +
      sizeof(LUID_AND_ATTRIBUTES));
  tp->PrivilegeCount = 1;
  if (!LookupPrivilegeValueA(NULL, "SeShutdownPrivilege",
      &tp->Privileges[0].Luid)) {
    Log::Error("LookupPrivilegeValue failed, error code is %d", GetLastError());
    CloseHandle(token_handle);
    return;
  }

  tp->Privileges[0].Attributes = SE_PRIVILEGE_ENABLED;
  if (!AdjustTokenPrivileges(token_handle, FALSE, tp, 0, NULL, NULL)) {
    CloseHandle(token_handle);
    return;
  }

  if (!InitiateSystemShutdownEx(NULL, NULL, 0, TRUE, bRebootAfterShutdown,
      SHTDN_REASON_FLAG_PLANNED | SHTDN_REASON_MAJOR_OTHER | SHTDN_REASON_MINOR_OTHER)) {
    Log::Error("InitiateSystemShutdownEx failed, error code is %d",
        GetLastError());
    // Log a message to the system log here about a failed shutdown
  }
  Log::Info("InitiateSystemShutdownEx succeeded");

  CloseHandle(token_handle);
}


DWORD XenThreadFunc(LPDWORD param) {
  HANDLE handle;
  int state = 0;
  char *path;
  char *buf;

  path = get_xen_interface_path();
  if (path == NULL)
    return FALSE;

  handle = CreateFileA(path, FILE_GENERIC_READ | FILE_GENERIC_WRITE, 0,
      NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);

  int ret = xb_add_watch(handle, "control/shutdown");

  while (!terminatingService && xb_wait_event(handle)) {
    buf = xb_read(handle, "control/shutdown");

    if (strcmp("poweroff", buf) == 0 || strcmp("halt", buf) == 0) {
      do_shutdown(FALSE);
    } else if (strcmp("reboot", buf) == 0) {
      do_shutdown(TRUE);
    } else if (strcmp("hibernate", buf) == 0) {
      do_hibernate();
    }
  }

  return TRUE;
}

// This function consolidates the activities of
// updating the service status with
// SetServiceStatus
BOOL SendStatusToSCM(DWORD dwCurrentState,
  DWORD dwWin32ExitCode,
  DWORD dwServiceSpecificExitCode,
  DWORD dwCheckPoint,
  DWORD dwWaitHint) {
  BOOL ret;
  SERVICE_STATUS serviceStatus;

  // Fill in all of the SERVICE_STATUS fields
  serviceStatus.dwServiceType = SERVICE_WIN32_OWN_PROCESS;
  serviceStatus.dwCurrentState = dwCurrentState;

  // If in the process of doing something, then accept
  // no control events, else accept anything
  if (dwCurrentState == SERVICE_START_PENDING) {
    serviceStatus.dwControlsAccepted = 0;
  } else {
    serviceStatus.dwControlsAccepted =
    SERVICE_ACCEPT_STOP |
    SERVICE_ACCEPT_PAUSE_CONTINUE |
    SERVICE_ACCEPT_SHUTDOWN;
  }

  // if a specific exit code is defined, set up
  // the win32 exit code properly
  if (dwServiceSpecificExitCode == 0) {
    serviceStatus.dwWin32ExitCode = dwWin32ExitCode;
  } else {
    serviceStatus.dwWin32ExitCode =
    ERROR_SERVICE_SPECIFIC_ERROR;
  }
  serviceStatus.dwServiceSpecificExitCode =
    dwServiceSpecificExitCode;

  serviceStatus.dwCheckPoint = dwCheckPoint;
  serviceStatus.dwWaitHint = dwWaitHint;

  // Pass the status record to the SCM
  ret = SetServiceStatus(serviceStatusHandle,
    &serviceStatus);

  return ret;
}

VOID ControlHandler(DWORD controlCode) {
  DWORD currentState = 0;
  BOOL ret;

  switch (controlCode) {
    // Stop the service
  case SERVICE_CONTROL_STOP:
    // Tell the SCM what's happening
    ret = SendStatusToSCM(SERVICE_STOP_PENDING,
      NO_ERROR, 0, 1, 5000);
    runningService = FALSE;
    // Set the event that is holding ServiceMain
    // so that ServiceMain can return
    // Set the event that is holding the UpdaterThreadFunc
    // so that the UpdaterThreadFunc can return
    PulseEvent(gTerminateEvent);
    terminatingService = TRUE;
    return;

    // Pause the service
  case SERVICE_CONTROL_PAUSE:
    if (runningService && !pauseService) {
      // Tell the SCM what's happening
      ret = SendStatusToSCM(
        SERVICE_PAUSE_PENDING,
        NO_ERROR, 0, 1, 1000);
      pauseService = TRUE;
      SuspendThread(producerThreadHandle);
      SuspendThread(consumerThreadHandle);
      SuspendThread(updaterThreadHandle);
      SuspendThread(serverSyncThreadHandle);
      SuspendThread(xenThreadHandle);
      SuspendThread(xenCmdReadThread);
      SuspendThread(xenCmdExecThread);
      currentState = SERVICE_PAUSED;
    }
    break;

    // Resume from a pause
  case SERVICE_CONTROL_CONTINUE:
    if (runningService && pauseService) {
      ret = SendStatusToSCM(
        SERVICE_CONTINUE_PENDING,
        NO_ERROR, 0, 1, 1000);
      pauseService = FALSE;
      ResumeThread(producerThreadHandle);
      ResumeThread(consumerThreadHandle);
      ResumeThread(updaterThreadHandle);
      ResumeThread(serverSyncThreadHandle);
      ResumeThread(xenThreadHandle);
      ResumeThread(xenCmdExecThread);
      ResumeThread(xenCmdReadThread);
      currentState = SERVICE_RUNNING;
    }
    break;

    // Update current status
  case SERVICE_CONTROL_INTERROGATE:
    // it will fall to bottom and send status
    break;

    // Do nothing in a shutdown. Could do cleanup
    // here but it must be very quick.
  case SERVICE_CONTROL_SHUTDOWN:
    return;

  default:
    break;
  }

  SendStatusToSCM(currentState, NO_ERROR, 0, 0, 0);
}

// Initializes the service by starting its threads
BOOL InitService() {
  Singleton<task_engine::TimerManager>::I().Start();
  Singleton<task_engine::TaskSchedule>::I().Fetch();
  Singleton<task_engine::TaskSchedule>::I().FetchPeriodTask();

  DWORD id;

  producerThreadHandle = CreateThread(0, 0,
    (LPTHREAD_START_ROUTINE)ProducerThreadFunc,
    0, 0, &id);

  if (producerThreadHandle == 0) {
    Log::Error("Failed to create the producer thread, error code is %d",
        GetLastError());
    return FALSE;
  }

  consumerThreadHandle = CreateThread(0, 0,
    (LPTHREAD_START_ROUTINE)ConsumerThreadFunc,
    0, 0, &id);
  if (consumerThreadHandle == 0) {
    Log::Error("Failed to create the consumer thread, error code is %d",
        GetLastError());
    return FALSE;
  }

  serverSyncThreadHandle = CreateThread(0, 0,
    (LPTHREAD_START_ROUTINE)ServerSyncThreadFunc,
    0, 0, &id);
  if (serverSyncThreadHandle == 0) {
    Log::Error("Failed to create the server sync thread, error code is %d",
        GetLastError());
    return FALSE;
  }

  xenThreadHandle = CreateThread(0, 0,
    (LPTHREAD_START_ROUTINE)XenThreadFunc,
    0, 0, &id);
  if (xenThreadHandle == 0) {
    Log::Error("Failed to create the xen thread, error code is %d",
        GetLastError());
    return FALSE;
  }

  updaterThreadHandle = CreateThread(0, 0,
    (LPTHREAD_START_ROUTINE)UpdaterThreadFunc,
    0, 0, &id);
  if (updaterThreadHandle == 0) {
    Log::Error("Failed to create the updater thread, error code is %d",
        GetLastError());
    return FALSE;
  }

  param.terminatingService = &terminatingService;
  param.kicker = []() {
    InterlockedIncrement(&gMsgCount);
  };

  XSShellStart(&param, xenCmdExecThread, xenCmdReadThread);

  runningService = TRUE;
  return TRUE;
}

// Handle service fatal error and close the service
VOID Terminate(DWORD errCode) {
  // Send a message to the scm to tell about stopage
  if (serviceStatusHandle) {
    SendStatusToSCM(SERVICE_STOPPED, errCode,
      0, 0, 0);
  }

  // if gTerminateEvent has been created, close it.
  if (gTerminateEvent) {
    CloseHandle(gTerminateEvent);
  }


  if (consumerThreadHandle) {
    CloseHandle(consumerThreadHandle);
  }

  if (producerThreadHandle) {
    CloseHandle(producerThreadHandle);
  }

  if (updaterThreadHandle) {
    CloseHandle(updaterThreadHandle);
  }

  if (serverSyncThreadHandle) {
    CloseHandle(serverSyncThreadHandle);
  }

  if (xenThreadHandle) {
    CloseHandle(xenThreadHandle);
  }

  if (xenCmdExecThread) {
    CloseHandle(xenCmdExecThread);
  }

  if (xenCmdReadThread) {
    CloseHandle(xenCmdReadThread);
  }

  ExitProcess(errCode);
}

void ServiceMain(int argc, char** argv) {
  BOOL ret;

  serviceStatusHandle = RegisterServiceCtrlHandler(
    serviceName,
    (LPHANDLER_FUNCTION)ControlHandler);
  if (serviceStatusHandle == (SERVICE_STATUS_HANDLE)0) {
    // Registering Control Handler failed
    Terminate(GetLastError());
    return;
  }

  // Notify SCM of progress
  ret = SendStatusToSCM(SERVICE_START_PENDING,
    NO_ERROR, 0, 1, 5000);
  if (!ret) {
    Terminate(GetLastError());
    return;
  }


  // Create the termination event
  gTerminateEvent = CreateEvent(0, TRUE, FALSE, 0);
  if (!gTerminateEvent) {
    Terminate(GetLastError());
    return;
  }


  // Notify SCM of progress
  ret = SendStatusToSCM(SERVICE_START_PENDING,
    NO_ERROR, 0, 2, 1000);
  if (!ret) {
    Terminate(GetLastError());
    return;
  }

  // Start the service itself
  ret = InitService();
  if (!ret) {
    Terminate(GetLastError());
    return;
  }

  // The service is now running.
  // Notify SCM of progress
  ret = SendStatusToSCM(SERVICE_RUNNING,
    NO_ERROR, 0, 0, 0);
  if (!ret) {
    Terminate(GetLastError());
    return;
  }

  // Wait for stop signal, and then terminate
  WaitForSingleObject(gTerminateEvent, INFINITE);

  Terminate((DWORD)0);
}

using optparse::OptionParser;

OptionParser& initParser() {
  static OptionParser parser = OptionParser().description("Aliyun Assist Copyright (c) 2017-2018 Alibaba Group Holding Limited");

  parser.add_option("-v", "--version")
    .dest("version")
    .action("store_true")
    .help("show version and exit");

  parser.add_option("-fetch_task", "--fetch_task")
    .action("store_true")
    .dest("fetch_task")
    .help("fetch tasks from server and run tasks");

  parser.add_option("-service", "--service")
    .action("store_true")
    .dest("service")
    .help("start as service");

  return parser;
}

void try_connect_again(void) {
  int index = 3;
  while(true) {
    Sleep(index*60*1000);
    if(index < 100) {
      index = index * 2;
    }
    AssistPath path_service("");
    HostChooser  host_choose;
    bool found = host_choose.Init(path_service.GetConfigPath());
    if (found) {
      break;
    }
  }
}

void main(int argc, char *argv[]) {
#if defined(_WIN32)
  SetDllDirectory(TEXT(""));
  DumpService::InitMinDump("aliyun service");
#endif
  AssistPath path_service("");
  std::string log_path = path_service.GetLogPath();
  log_path += FileUtils::separator();
  log_path += "aliyun_assist_main.log";
  Log::Initialise(log_path);
  Log::Info("main begin...");

  OptionParser& parser = initParser();
  optparse::Values options = parser.parse_args(argc, argv);

  if (options.is_set("version")) {
    printf("%s\n", FILE_VERSION_RESOURCE_STR);
    return;
  }
  curl_global_init(CURL_GLOBAL_ALL);
  HostChooser  host_choose;
  bool found = host_choose.Init(path_service.GetConfigPath());
  if (!found) {
    new std::thread(try_connect_again);
    Log::Error("could not find a match region host");
  }
  if (options.is_set("service")) {
    SERVICE_TABLE_ENTRY serviceTable[] = {
      {
        serviceName,
        (LPSERVICE_MAIN_FUNCTION)ServiceMain
      },
      {
        NULL,
        NULL
      }
    };

    BOOL ret;
    DWORD errCode;

    // Register with the SCM
    ret = StartServiceCtrlDispatcher(serviceTable);
    if (!ret) {
      errCode = GetLastError();
      Log::Error("Start Service Ctrl Dispatcher Failure, error code is %d",
          errCode);
      Terminate(errCode);
    }

  } else if (options.is_set("fetch_task")) {
    Singleton<task_engine::TaskSchedule>::I().Fetch();
    Singleton<task_engine::TaskSchedule>::I().FetchPeriodTask();
    Sleep(3600*1000);
    return;
  }

  curl_global_cleanup();
  parser.print_help();
}
