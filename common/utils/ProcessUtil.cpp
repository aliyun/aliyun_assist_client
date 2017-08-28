// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "ProcessUtil.h"

#include "Log.h"

#include <string.h>
#include <vector>
#include <iostream>
#include <algorithm>

#if defined _WIN32
#include <windows.h>
#else
#include <stdlib.h>
#include <sys/wait.h>
#include <errno.h>
#include <stdio.h>
#include <string.h>

#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <sys/file.h>
#include <unistd.h>
#endif

#if !defined _WIN32
#include <sys/sysctl.h>
#include <sys/types.h>
#endif

int ProcessUtils::runSync(const std::string& executable, const std::string& args) {
#if !defined _WIN32
  return runSyncUnix(executable, args);
#else
  return runWindows(executable, args);
#endif
}

#if !defined(_WIN32)
static int g_single_proc_inst_lock_fd = -1;

static void single_proc_inst_lockfile_cleanup(void) {
  if (g_single_proc_inst_lock_fd != -1) {
    close(g_single_proc_inst_lock_fd);
    g_single_proc_inst_lock_fd = -1;
  }
}

bool ProcessUtils::is_single_proc_inst_running(const char *process_name) {
  char lock_file[128];
  snprintf(lock_file, sizeof(lock_file), "/var/tmp/%s.lock", process_name);

  g_single_proc_inst_lock_fd = open(lock_file, O_CREAT | O_RDWR, 0644);
  if (-1 == g_single_proc_inst_lock_fd) {
    Log::Error("Fail to open lock file(%s). Error: %s\n",
      lock_file, strerror(errno));
    return false;
  }

  if (0 == flock(g_single_proc_inst_lock_fd, LOCK_EX | LOCK_NB)) {
    atexit(single_proc_inst_lockfile_cleanup);
    return true;
  }

  close(g_single_proc_inst_lock_fd);
  g_single_proc_inst_lock_fd = -1;
  return false;
}
#endif

#if defined _WIN32
namespace {
std::string toWindowsPathSeparators(const std::string& str) {
  std::string result = str;
  std::replace(result.begin(), result.end(), '/', '\\');
  return result;
}
}
int ProcessUtils::runWindows(const std::string& _executable,
                             const std::string& _args) {
  // most Windows API functions allow back and forward slashes to be
  // used interchangeably.  However, an application started with
  // CreateProcess() may fail to find Side-by-Side library dependencies
  // in the same directory as the executable if forward slashes are
  // used as path separators, so convert the path to use back slashes here.
  //
  // This may be related to LoadLibrary() requiring backslashes instead
  // of forward slashes.
  std::string executable = toWindowsPathSeparators(_executable);

  std::string commandLine = _args;

  STARTUPINFOA startupInfo;
  ZeroMemory(&startupInfo, sizeof(startupInfo));
  startupInfo.cb = sizeof(startupInfo);

  PROCESS_INFORMATION processInfo;
  ZeroMemory(&processInfo, sizeof(processInfo));

  char* commandLineStr = _strdup(commandLine.c_str());
  bool result = CreateProcessA(
                  executable.c_str(),
                  commandLineStr,
                  0 /* process attributes */,
                  0 /* thread attributes */,
                  false /* inherit handles */,
                  NORMAL_PRIORITY_CLASS /* creation flags */,
                  0 /* environment */,
                  0 /* current directory */,
                  &startupInfo /* startup info */,
                  &processInfo /* process information */
                );

  if (!result) {
    Log::Error("Failed to start child process. ");
    return false;
  } else {
    if (WaitForSingleObject(processInfo.hProcess, INFINITE) == WAIT_OBJECT_0) {
      DWORD status = 0;
      if (GetExitCodeProcess(processInfo.hProcess, &status) != 0) {
        Log::Error("Failed to get exit code for process");
      }
      return true;
    } else {
      Log::Error("Failed to wait for process to finish");
      return false;
    }
  }
}
#endif

#if !defined _WIN32
int ProcessUtils::runSyncUnix(const std::string& executable,
                              const std::string& args) {
  PLATFORM_PID pid = runAsyncUnix(executable, args);
  int status = 0;
  if (waitpid(pid, &status, 0) != -1) {
    if (WIFEXITED(status)) {
      return true;
    } else {
      Log::Warn("Child exited abnormally");
      return false;
    }
  } else {
    Log::Warn("Failed to get exit status of child ");
    return false;
  }
}

PLATFORM_PID ProcessUtils::runAsyncUnix(const std::string& executable,
                                        const std::string& args) {
  pid_t child = fork();
  if (child == 0) {
    char * argv[2] = {0};
    argv[0] = (char* )args.c_str();
    if (execvp(executable.c_str(), argv) == -1) {
      Log::Error("error start child");
    }
    exit(0);
  }
  return child;
}
#endif
