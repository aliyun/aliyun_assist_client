#if defined(_WIN32)
#include "dump.h"

#include "windows.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "jsoncpp/json.h"
#include "utils/http_request.h"
#include "utils/CheckNet.h"
#include "utils/OsVersion.h"
#include "dbghelp.h"
#pragma comment(lib, "dbghelp.lib")

inline BOOL IsDataSectionNeeded(const WCHAR* pModuleName) {
  if (pModuleName == 0) {
    return FALSE;
  }

  WCHAR szFileName[_MAX_FNAME] = L"";
  _wsplitpath(pModuleName, NULL, NULL, szFileName, NULL);

  if (wcsicmp(szFileName, L"ntdll") == 0)
    return TRUE;

  return FALSE;
}

inline BOOL CALLBACK MiniDumpCallback(PVOID                            pParam,
                                      const PMINIDUMP_CALLBACK_INPUT   pInput,
                                      PMINIDUMP_CALLBACK_OUTPUT        pOutput) {
  if (pInput == 0 || pOutput == 0)
    return FALSE;

  switch (pInput->CallbackType) {
  case ModuleCallback:
    if (pOutput->ModuleWriteFlags & ModuleWriteDataSeg)
      if (!IsDataSectionNeeded(pInput->Module.FullPath))
        pOutput->ModuleWriteFlags &= (~ModuleWriteDataSeg);
  case IncludeModuleCallback:
  case IncludeThreadCallback:
  case ThreadCallback:
  case ThreadExCallback:
    return TRUE;
  default:
    ;
  }

  return FALSE;
}

inline void CreateMiniDump(PEXCEPTION_POINTERS pep, std::string strFileName) {
  HANDLE hFile = CreateFileA(strFileName.c_str(), GENERIC_READ | GENERIC_WRITE,
                             FILE_SHARE_WRITE, NULL, CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, NULL);

  if ((hFile != NULL) && (hFile != INVALID_HANDLE_VALUE)) {
    MINIDUMP_EXCEPTION_INFORMATION mdei;
    mdei.ThreadId = GetCurrentThreadId();
    mdei.ExceptionPointers = pep;
    mdei.ClientPointers = NULL;

    MINIDUMP_CALLBACK_INFORMATION mci;
    mci.CallbackRoutine = (MINIDUMP_CALLBACK_ROUTINE)MiniDumpCallback;
    mci.CallbackParam = 0;

    ::MiniDumpWriteDump(::GetCurrentProcess(), ::GetCurrentProcessId(), hFile, MiniDumpNormal, (pep != 0) ? &mdei : 0, NULL, &mci);

    CloseHandle(hFile);
  }
}
namespace {
std::string app_name;
std::string get_request_string() {
  Json::Value jsonRoot;
#ifdef _WIN32
  jsonRoot["os"] = "windows";
#else
  jsonRoot["os"] = "linux";
#endif
  jsonRoot["os_version"] = OsVersion::GetVersion();
  jsonRoot["appName"] = app_name;

  return jsonRoot.toStyledString();
}

void Report() {
  std::string json = get_request_string();
  std::string response;
  if (HostChooser::m_HostSelect.empty()) {
    return;
  }
  std::string url = "http://" + HostChooser::m_HostSelect;
  url += "/luban/api/v1/exception/dump_report";
  HttpRequest::http_request_post(url, json, response);
}
}  // namespace
LONG __stdcall MyUnhandledExceptionFilter(PEXCEPTION_POINTERS pExceptionInfo) {
  AssistPath path_service("");
  std::string log_path = path_service.GetLogPath();
  std::string file_name;
  char string[1024] = { 0 };
  itoa(::GetTickCount(), string, 10);
  file_name = log_path + FileUtils::separator() + string + ".dmp";
  CreateMiniDump(pExceptionInfo, file_name);

  return EXCEPTION_EXECUTE_HANDLER;
}

// 此函数一旦成功调用，之后对 SetUnhandledExceptionFilter 的调用将无效
void DisableSetUnhandledExceptionFilter() {
  void* addr = (void*)GetProcAddress(LoadLibraryA("kernel32.dll"),
                                     "SetUnhandledExceptionFilter");

  if (addr) {
    unsigned char code[16];
    int size = 0;

    code[size++] = 0x33;
    code[size++] = 0xC0;
    code[size++] = 0xC2;
    code[size++] = 0x04;
    code[size++] = 0x00;

    DWORD dwOldFlag, dwTempFlag;
    VirtualProtect(addr, size, PAGE_READWRITE, &dwOldFlag);
    WriteProcessMemory(GetCurrentProcess(), addr, code, size, NULL);
    VirtualProtect(addr, size, dwOldFlag, &dwTempFlag);
  }
}

void DumpService::InitMinDump(std::string product_name) {
  //注册异常处理函数
  app_name = product_name;
  SetUnhandledExceptionFilter(MyUnhandledExceptionFilter);

  //使SetUnhandledExceptionFilter
  DisableSetUnhandledExceptionFilter();
}
#endif

