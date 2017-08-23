// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "./DeviceUtils.h"

#ifdef _WIN32
#include <Windows.h>
#include <setupapi.h>
#else
#include <sys/utsname.h>
#endif  // _WIN32

bool DeviceUtils::IsXen() {
#ifdef _WIN32
  return WindowsIsXen();
#else
  return LinuxIsXen();
#endif
}

#ifdef _WIN32
bool DeviceUtils::WindowsIsXen() {
  HDEVINFO hdev;
  hdev = SetupDiGetClassDevs(0, 0, 0, DIGCF_ALLCLASSES | DIGCF_PRESENT);
  if (hdev == INVALID_HANDLE_VALUE) {
    return false;
  }

  SP_DEVINFO_DATA devinfo;
  devinfo.cbSize = sizeof(devinfo);

  for (int i = 0; SetupDiEnumDeviceInfo(hdev, i, &devinfo); i++) {
    DWORD  type;
    DWORD  len;

    char   _devid[512] = { 0 };
    len = sizeof(_devid) - 2;

    if (!SetupDiGetDeviceRegistryProperty(
      hdev,
      &devinfo,
      SPDRP_HARDWAREID,
      &type,
      (PBYTE)_devid,
      len,
      &len)) {
      continue;
    }

    if (strstr(_devid, "xen\\")) {
      return true;
    }
  }
  return false;
}
#endif  // _WIN32

#ifndef _WIN32
bool DeviceUtils::LinuxIsXen() {
  return false;
}
#endif  // !_WIN32
