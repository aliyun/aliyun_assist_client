// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "./xs_shell.h"
#include "./xs.h"
#include "utils/Log.h"
#include "utils/CheckNet.h"

HANDLE gMutexStdin;
HANDLE gEvent;

CMDStatus gCMDStatus = CMD_STATUS_STOPPED;
HANDLE gProcessRunning = NULL;
DWORD gProcessID = 0;
char cmdBuf[CMD_MAX_LENGTH + LENGTH_TIMESTAMP + 1];

void WriteToXenstore(HANDLE handle,
    char* path,
    char* buf,
    size_t bufLen,
    char* ptimeStamp) {
  char writeBuf[BUFFER_SIZE + LENGTH_TIMESTAMP];
  size_t str_len;

  if (ptimeStamp != NULL) {
    if (strlen(ptimeStamp) >= LENGTH_TIMESTAMP)
      memcpy_s(writeBuf, BUFFER_SIZE + LENGTH_TIMESTAMP,
          ptimeStamp, LENGTH_TIMESTAMP);
    memcpy_s(writeBuf + LENGTH_TIMESTAMP, BUFFER_SIZE, buf, bufLen);
    str_len = bufLen + LENGTH_TIMESTAMP;
  } else {
    memcpy_s(writeBuf, BUFFER_SIZE + LENGTH_TIMESTAMP, buf, bufLen);
    str_len = bufLen;
  }

  Log::Info("xs_write: [%s] [%.*s] [%d]", path, str_len, writeBuf, str_len);
  xb_write(handle, path, writeBuf, str_len);
  return;
}

void TerminateSubProcess() {
  PROCESSENTRY32 proc_struct;
  HANDLE snapshot;
  DWORD ProcessIDFind = 0;
  HANDLE hProcessHandle;

  proc_struct.dwSize = sizeof(PROCESSENTRY32);
  snapshot = CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0);

  if (Process32First(snapshot, &proc_struct) == TRUE) {
    while (Process32Next(snapshot, &proc_struct)) {
      if (proc_struct.th32ParentProcessID == gProcessID) {
        ProcessIDFind = proc_struct.th32ProcessID;
        break;
      }
    }
  }

  if (ProcessIDFind) {
    hProcessHandle = OpenProcess(PROCESS_ALL_ACCESS, FALSE, ProcessIDFind);
    if (hProcessHandle != NULL)
      TerminateProcess(hProcessHandle, 0);
  }
}

void ExecCmd(HANDLE handle, XENKICKER kicker) {
  char* ptimeStamp = EMPTY_TIMESTAMP;
  char* pCmdline = NULL;

  /*check*/
  Log::Info("cmdBuf = %s", cmdBuf);

  if (strlen(cmdBuf) <= LENGTH_TIMESTAMP) {
    WriteToXenstore(handle, XS_PATH_CMDSTDOUT, ERR_CMD_IS_EMPTY,
        strlen(ERR_CMD_IS_EMPTY), ptimeStamp);
    return;
  }

  ptimeStamp = cmdBuf;

  char* strCmd = strstr(cmdBuf, "kick_vm");
  if ((strCmd != NULL) && !strcmp(strCmd, "kick_vm")) {
    kicker();
    if (HostChooser::m_Classical) {
      WriteToXenstore(handle, XS_PATH_CMDSTDOUT, SUC_KICK_VM_CLASSICAL,
          strlen(SUC_KICK_VM_CLASSICAL), ptimeStamp);
      Log::Info("kick under classical net");
    }
    else {
      WriteToXenstore(handle, XS_PATH_CMDSTDOUT, SUC_KICK_VM,
          strlen(SUC_KICK_VM), ptimeStamp);
      Log::Info("kick under vpc net");
    }
  } else {
    WriteToXenstore(handle, XS_PATH_CMDSTDOUT, ERR_CMD_NOT_SUPPORT,
        strlen(ERR_CMD_NOT_SUPPORT), ptimeStamp);
  }
}

/*Execute the cmd and write output to xenstore*/
unsigned __stdcall CmdExecThreadProc(void* pArguments) {
  HANDLE handle;
  char *path = NULL;
  th_param *pargs;
  pargs = reinterpret_cast<th_param*>(pArguments);

  Log::Info("ExecThreadProc Start");

  if ((path = get_xen_interface_path()) == NULL)
    return 0;

  handle = CreateFileA(path, FILE_GENERIC_READ | FILE_GENERIC_WRITE, 0,
      NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);

  while (!(*pargs->terminatingService)) {
    WaitForSingleObject(gEvent, INFINITE);

    Log::Info("Exec Start");
    ExecCmd(handle, pargs->kicker);
    Log::Info("Exec Done");

    ResetEvent(gEvent);

    WaitForSingleObject(gMutexStdin, INFINITE);
    gCMDStatus = CMD_STATUS_STOPPED;
    ReleaseMutex(gMutexStdin);
  }
  return 0;
}

/*Check State Proc*/
unsigned __stdcall CmdCheckThreadProc(void* pArguments) {
  HANDLE watch_handle;
  HANDLE handle;
  char *path = NULL;
  char *pargs;
  pargs = reinterpret_cast<char*>(pArguments);

  Log::Info("CheckThreadProc Start");

  if ((path = get_xen_interface_path()) == NULL)
    return 0;

  watch_handle = CreateFileA(path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
      0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
  xb_add_watch(watch_handle, XS_PATH_CMDSTATEIN);

  handle = CreateFileA(path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
      0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
  WriteToXenstore(handle, XS_PATH_CMDSTATEOUT, STATE_ENABLE,
      strlen(STATE_ENABLE), NULL);

  while (xb_wait_event(watch_handle)) {
    WriteToXenstore(handle, XS_PATH_CMDSTATEOUT, STATE_ENABLE,
        strlen(STATE_ENABLE), NULL);
  }

  Log::Info("CheckThreadProc End");
  return 0;
}

/*Command Receive Proc*/
unsigned __stdcall CmdReadThreadProc(void* pArguments) {
  HANDLE watch_handle;
  HANDLE handle;
  HANDLE hCmdCheckThread;
  char *path;
  char *buf;
  BOOL *terminatingService;
  unsigned threadID;

  terminatingService = reinterpret_cast<BOOL*>(pArguments);
  Log::Info("ReadThreadProc Start");

  if ((path = get_xen_interface_path()) == NULL)
    return 0;

  watch_handle = CreateFileA(path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
      0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
  xb_add_watch(watch_handle, XS_PATH_CMDSTDIN);

  hCmdCheckThread = (HANDLE)_beginthreadex(NULL, 0, &CmdCheckThreadProc,
      NULL, 0, &threadID);
  if (hCmdCheckThread == NULL) {
    return 0;
  }

  handle = CreateFileA(path, FILE_GENERIC_READ | FILE_GENERIC_WRITE,
      0, NULL, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);

  while (!(*terminatingService) && xb_wait_event(watch_handle)) {
    buf = xb_read(handle, XS_PATH_CMDSTDIN);

    if (buf == NULL)
      continue;

    Log::Info("new event: %s", buf);

    if (!memcmp(buf, SHELL_CMD_TERM_PROCESS, strlen(SHELL_CMD_TERM_PROCESS))) {
      TerminateSubProcess();
      TerminateProcess(gProcessRunning, 0);
      free(buf);
      continue;
    }

    WaitForSingleObject(gMutexStdin, INFINITE);

    if (gCMDStatus == CMD_STATUS_RUNNING) {
      if (memcmp(cmdBuf, buf, strlen(buf))) {
        WriteToXenstore(handle, XS_PATH_CMDSTDOUT, ERR_CMD_LAST_IS_RUNNING,
            strlen(ERR_CMD_LAST_IS_RUNNING), buf);
      }

      ReleaseMutex(gMutexStdin);
      free(buf);

      continue;
    }

    gCMDStatus = CMD_STATUS_RUNNING;
    strcpy_s(cmdBuf, CMD_MAX_LENGTH + LENGTH_TIMESTAMP, buf);
    cmdBuf[CMD_MAX_LENGTH] = '\0';

    SetEvent(gEvent);
    ReleaseMutex(gMutexStdin);
    free(buf);
  }

  Log::Info("ReadThreadProc End");
  return 0;
}

int XSShellStart(th_param* param,
    HANDLE& hCmdExecThread,
    HANDLE& hCmdReadThread) {
  unsigned threadID;

  HANDLE hToken;
  LUID seDebug;
  TOKEN_PRIVILEGES tkp;

  Log::Info("gshell start");

  /*upgrade privilege for kill subprocess*/
  OpenProcessToken(GetCurrentProcess(),
      TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &hToken);
  LookupPrivilegeValue(NULL, SE_DEBUG_NAME, &seDebug);

  tkp.PrivilegeCount = 1;
  tkp.Privileges[0].Luid = seDebug;
  tkp.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED;

  AdjustTokenPrivileges(hToken, FALSE, &tkp, sizeof(tkp), NULL, NULL);
  CloseHandle(hToken);

  gEvent = CreateEvent(NULL, TRUE, FALSE, NULL);
  gMutexStdin = CreateMutex(NULL, FALSE, NULL);

  /*Start xenstore shell interface*/
  hCmdExecThread = (HANDLE)_beginthreadex(NULL, 0, &CmdExecThreadProc,
      param, 0, &threadID);
  if (hCmdExecThread == NULL) {
    Log::Error("CmdExecThreadProc create fail: %d", GetLastError());
    return 0;
  }
  hCmdReadThread = (HANDLE)_beginthreadex(NULL, 0, &CmdReadThreadProc,
      param->terminatingService, 0, &threadID);
  if (hCmdReadThread == NULL) {
    Log::Error("CmdReadThreadProc create fail: %d", GetLastError());
    return 0;
  }

  Log::Info("Threads created");
  return 1;
}

