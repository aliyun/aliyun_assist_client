// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "./gshell.h"

#include <string>
#include <windows.h>
#include "utils/Log.h"

Gshell::Gshell(KICKER kicker) {
  m_kicker = kicker;

  m_hFile = CreateFileA("\\\\.\\Global\\org.qemu.guest_agent.0",
      GENERIC_ALL,
      0,
      0,
      OPEN_EXISTING,
      FILE_ATTRIBUTE_NORMAL,
      NULL);

  if ( m_hFile != INVALID_HANDLE_VALUE ) {
      return;
  }
  Log::Error("open org.qemu.guest_agent.0 failed:%d", GetLastError());
  m_hFile = CreateFileA("\\\\.\\COM1",
      GENERIC_ALL,
      0,
      0,
      OPEN_EXISTING,
      FILE_ATTRIBUTE_NORMAL,
      NULL);

  if (m_hFile == INVALID_HANDLE_VALUE) {
    return;
  }

  COMMTIMEOUTS comTimeOut = { 0 };
  comTimeOut.ReadIntervalTimeout = 1;
  SetCommTimeouts(m_hFile, &comTimeOut);

}

Gshell::~Gshell() {
  if ( m_hFile != NULL && m_hFile != INVALID_HANDLE_VALUE ) {
    CloseHandle(m_hFile);
    m_hFile = NULL;
  }
}

bool  Gshell::Poll() {
  if ( m_hFile == NULL || m_hFile == INVALID_HANDLE_VALUE ) {
      return false;
    }

  char  buffer[0x1000] = {0};
  DWORD len = 0;
  BOOL  ret = FALSE;

  ret = ReadFile(m_hFile, buffer, sizeof(buffer) - 1, &len, 0);
  if ( !ret || len == 0 ) {
      Sleep(THREAD_SLEEP_TIME);
      return true;
  }
  buffer[len] = 0;

#ifdef _DEBUG
  printf("[r]:%s\n", buffer);
#endif

  string output;
  Parse(buffer, output);
  Log::Info("[w]:%s\n", output.c_str());
  WriteFile(m_hFile, output.c_str(), output.length(), &len, 0);

#ifdef _DEBUG
  printf("[w]:%s\n", output.c_str());
#endif

  return true;
}

void  Gshell::Parse(string input, string& output) {
	Log::Info("command:%s", input.c_str());
  string errinfo;
  auto json = json11::Json::parse(input, errinfo);
  if ( errinfo != "" ) {
    return;
  }

  if (json["execute"] == "guest-sync") {
      return QmpGuestSync(json["arguments"], output);
  }

  if (json["execute"] == "guest-command") {
      return QmpGuestCommand(json["arguments"], output);
  }

  if (json["execute"] == "guest-shutdown") {
    return QmpGuestShutdown(json["arguments"], output);
  }

  Error err;
  err.SetDesc("not suport");
  output = err.Json().dump() + "\n";
}

// gshell check ready

/*{ 'command': 'guest-sync',
'data' : { 'id': 'int' },
'returns' : 'int' }*/
void Gshell::QmpGuestSync(json11::Json  arguments, string& output) {
    json11::Json resp = json11::Json::object{ { "return", arguments["id"] } };
    output = resp.dump() + "\n";
}

/*
{ 'command': 'guest-command',
'data': { 'cmd': 'str', 'timeout': 'int' },
'returns': 'GuestCommandResult' }

{ 'type': 'GuestCommandResult',
'data': { 'result': 'int', 'cmd_output': 'str' } }
*/

void  Gshell::QmpGuestCommand(json11::Json  arguments, string& output) {
  string cmd = arguments["cmd"].string_value();
  if (arguments["cmd"] == "kick_vm" && m_kicker) {
    m_kicker();
    json11::Json   GuestCommandResult = json11::Json::object{
        { "result",8 },
        { "cmd_output", "execute kick_vm success" }
    };

    json11::Json  resp = json11::Json::object{ { "return",
        GuestCommandResult } };
    output = resp.dump() + "\n";
  } else {
    Error err;
    err.SetDesc("not suport");
    output = err.Json().dump() + "\n";
  }
}

bool Gshell::EnablePrivilege(const char *name, Error& errp) {
  HANDLE token = NULL;

  if (!OpenProcessToken(GetCurrentProcess(),
      TOKEN_ADJUST_PRIVILEGES | TOKEN_QUERY, &token)) {
    Log::Info("OpenProcessToken failed : %d", GetLastError());
    errp.SetDesc("failed to open privilege token");
    return false;
  }

  TOKEN_PRIVILEGES priv;
  if (!LookupPrivilegeValueA(NULL, name, &priv.Privileges[0].Luid)) {
    Log::Info("LookupPrivilegeValueA failed : %d", GetLastError());
    errp.SetDesc("no luid for requested privilege");
    CloseHandle(token);
    return false;
  }

  priv.PrivilegeCount = 1;
  priv.Privileges[0].Attributes = SE_PRIVILEGE_ENABLED;

  if (!AdjustTokenPrivileges(token, FALSE, &priv, 0, NULL, 0)) {
    Log::Info("AdjustTokenPrivileges failed : %d", GetLastError());
    errp.SetDesc("unable to acquire requested privilege");
    CloseHandle(token);
    return false;
  }

  CloseHandle(token);
  return true;
}

void  Gshell::QmpGuestShutdown(json11::Json arguments, string& output) {
  Error err;
  BOOL  bRebootAfterShutdown;

  if ( arguments["mode"].is_null() ) {
    err.SetDesc("powerdown|reboot");
    output = err.Json().dump() + "\n";
    return;
  }

  if (arguments["mode"].string_value() == "powerdown") {
    bRebootAfterShutdown = false;
  } else if (arguments["mode"].string_value() == "reboot") {
    bRebootAfterShutdown = true;
  } else {
    err.SetDesc("powerdown|reboot");
    output = err.Json().dump() + "\n";
    return;
  }

  if ( !EnablePrivilege("SeShutdownPrivilege", err) ) {
    output = err.Json().dump() + "\n";
    return;
  }

  if (!InitiateSystemShutdownEx(NULL,
      NULL,
      0,
      TRUE,
      bRebootAfterShutdown,
      SHTDN_REASON_FLAG_PLANNED |
      SHTDN_REASON_MAJOR_OTHER |
      SHTDN_REASON_MINOR_OTHER) ) {
      err.SetDesc("InitiateSystemShutdownEx fail");
      output = err.Json().dump() + "\n";
      Log::Info("InitiateSystemShutdownEx failed : %d", GetLastError());
  } else {
    json11::Json   GuestCommandResult = json11::Json::object{
        { "result", 8},
        { "cmd_output", "execute command success"}
    };
    json11::Json resp = json11::Json::object{ { "return",
        GuestCommandResult } };
    output = resp.dump() + "\n";
  }
}

