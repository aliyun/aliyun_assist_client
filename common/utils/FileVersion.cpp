#ifdef _WIN32
#include <Windows.h>
#else
#include <sys/utsname.h>
#endif // _WIN32

#include "FileVersion.h"

string FileVersion::GetFileVersion() {
#ifdef _WIN32
  return WindowsGetFileVersion();
#else
  return LinuxGetFileVersion();
#endif
};

#ifdef _WIN32
string FileVersion::WindowsGetFileVersion() {
  DWORD dummy;
  char ctemp[1024] = { 0 };
  GetModuleFileNameA(NULL, ctemp, 1024);
  const DWORD length = ::GetFileVersionInfoSizeA(ctemp, &dummy);
  if (length == 0)
    return "";

  char version[1024] = { 0 };

  if (!::GetFileVersionInfoA(ctemp, dummy, length, version))
    return "";
  return version;
}
#endif // _WIN32

#ifndef _WIN32
string FileVersion::LinuxGetFileVersion() {
  string version = "1.0.0.1";
  return version;
}
#endif // !_WIN32
