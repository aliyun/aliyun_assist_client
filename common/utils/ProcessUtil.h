// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#pragma once

#include <list>
#include <string>

#if !defined _WIN32
#include <unistd.h>
#define PLATFORM_PID pid_t
#else
#define PLATFORM_PID DWORD
#endif

class ProcessUtils {
 public:
  static int runSync(const std::string& executable,
                     const std::string& args);
#if !defined(_WIN32)
  static bool is_single_proc_inst_running(const char * process_name);
#endif

 private:
#if defined _WIN32
  static int runWindows(const std::string& executable,
                        const std::string& args);
#else
  static int runSyncUnix(const std::string& executable,
                         const std::string& args);
  static PLATFORM_PID runAsyncUnix(const std::string& executable,
                                   const std::string& args);
#endif
};
