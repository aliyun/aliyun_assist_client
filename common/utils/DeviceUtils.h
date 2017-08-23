// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#ifndef COMMON_UTILS_DEVICEUTILS_H_
#define COMMON_UTILS_DEVICEUTILS_H_

#include <string>
using std::string;

class  DeviceUtils {
 public:
  static bool IsXen();
 private:
  static bool WindowsIsXen();
  static bool LinuxIsXen();
};

#endif  // COMMON_UTILS_DEVICEUTILS_H_
