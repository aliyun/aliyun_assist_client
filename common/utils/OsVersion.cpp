#ifdef _WIN32
#include <Windows.h>
#include <stdio.h>
#include <tchar.h>
#include <conio.h>
#else
#include <sys/utsname.h>
#include <string.h>
#endif // _WIN32

#include "OsVersion.h"

string OsVersion::GetVersion() {
#ifdef _WIN32
  return WindowsGetVersion();
#else
  return LinuxGetVersion();
#endif
};

bool OsVersion::Is64BitOS() {
#ifdef _WIN32
  return WindowsIs64BitOS();
#else
  return LinuxIs64BitOS();
#endif
};

#ifdef _WIN32
string OsVersion::WindowsGetVersion() {

  SYSTEM_INFO info;
  GetSystemInfo(&info);
  OSVERSIONINFOEX os;
  os.dwOSVersionInfoSize = sizeof(OSVERSIONINFOEX);
  string osname = "unknown OperatingSystem.";

  if (GetVersionEx((OSVERSIONINFO *)&os)) {
    switch (os.dwMajorVersion) {
    case 4:
      switch (os.dwMinorVersion) {
      case 0:
        if (os.dwPlatformId == VER_PLATFORM_WIN32_NT)
          osname = "Microsoft Windows NT 4.0";
        else if (os.dwPlatformId == VER_PLATFORM_WIN32_WINDOWS)
          osname = "Microsoft Windows 95";
        break;
      case 10:
        osname = "Microsoft Windows 98";
        break;
      case 90:
        osname = "Microsoft Windows Me";
        break;
      }
      break;

    case 5:
      switch (os.dwMinorVersion) {
      case 0:
        osname = "Microsoft Windows 2000";
        break;

      case 1:
        osname = "Microsoft Windows XP";
        break;

      case 2:
        if (os.wProductType == VER_NT_WORKSTATION
            && info.wProcessorArchitecture == PROCESSOR_ARCHITECTURE_AMD64) {
          osname = "Microsoft Windows XP Professional x64 Edition";
        } else if (GetSystemMetrics(SM_SERVERR2) == 0)
          osname = "Microsoft Windows Server 2003";
        else if (GetSystemMetrics(SM_SERVERR2) != 0)
          osname = "Microsoft Windows Server 2003 R2";
        break;
      }
      break;

    case 6:
      switch (os.dwMinorVersion) {
      case 0:
        if (os.wProductType == VER_NT_WORKSTATION)
          osname = "Microsoft Windows Vista";
        else
          osname = "Microsoft Windows Server 2008";
        break;
      case 1:
        if (os.wProductType == VER_NT_WORKSTATION)
          osname = "Microsoft Windows 7";
        else
          osname = "Microsoft Windows Server 2008 R2";
        break;
      case 2:
        if (os.wProductType == VER_NT_WORKSTATION)
          osname = "Microsoft Windows 8";
        else
          osname = "Microsoft Windows Server 2012";
        break;
      case 3:
        if (os.wProductType == VER_NT_WORKSTATION)
          osname = "Microsoft Windows 8.1";
        else
          osname = "Microsoft Windows Server 2012 R2";
        break;
      }
      break;

    case 10:
      switch (os.dwMinorVersion) {
      case 0:
        if (os.wProductType == VER_NT_WORKSTATION)
          osname = "Microsoft Windows 10";
        else
          osname = "Microsoft Windows Server 2016 Technical Preview";//?????
        break;
      }
      break;
    }
  }
  // https://msdn.microsoft.com/en-us/library/ms724832.aspx
  return osname;
}

bool OsVersion::WindowsIs64BitOS()
{
  SYSTEM_INFO si;
  typedef VOID(WINAPI *LPFN_GetNativeSystemInfo)(LPSYSTEM_INFO lpSystemInfo);
  LPFN_GetNativeSystemInfo fnGetNativeSystemInfo = 
      (LPFN_GetNativeSystemInfo)GetProcAddress(GetModuleHandle(_T("kernel32")), "GetNativeSystemInfo");

  if (NULL != fnGetNativeSystemInfo) {
    fnGetNativeSystemInfo(&si);
  } else {
    GetSystemInfo(&si);
  }

  if (si.wProcessorArchitecture == PROCESSOR_ARCHITECTURE_AMD64 ||
    si.wProcessorArchitecture == PROCESSOR_ARCHITECTURE_IA64) {
    return true;
  }

  return false;
}

#endif // _WIN32

#ifndef _WIN32
string OsVersion::LinuxGetVersion() {
  string osname;
  struct utsname utsn;
  if (uname(&utsn)) {
    perror("uname");
    return osname;
  }

  osname += utsn.sysname;
  osname += "_";
  osname += utsn.version;
  osname += "_";
  osname += utsn.machine;

  //printf("sysname:%s\n", utsn.sysname);
  //printf("nodename:%s\n", utsn.nodename);
  //printf("release:%s\n", utsn.release);
  //printf("version:%s\n", utsn.version);
  //printf("machine:%s\n", utsn.machine);

  /*sysname:Linux
  nodename:ubuntu
  release:4.8.0-41-generic
  version:#44~16.04.1-Ubuntu SMP Fri Mar 3 17:11:16 UTC 2017
  machine:x86_64
  sh: 1: /bin: Permission denied
  */

  return osname;
}

bool OsVersion::LinuxIs64BitOS() {
  struct utsname utsn;
  if (uname(&utsn)) {
    return false;
  }

  if (strcmp(utsn.machine, "x86_64") == 0) {
    return true;
  } else {
    return false;
  }
}
#endif // !_WIN32
