#ifndef LOG_H
#define LOG_H

#include <string.h>
#include <vector>
#include <stdio.h>
#include <iostream>
#include <fstream>
#include <stdarg.h>
#define  ELPP_THREAD_SAFE
#define  ELPP_NO_DEFAULT_LOG_FILE
#include "easyloggingpp/easylogging++.h"

class Log {
 public:
  enum Type {
    LOG_TYPE_FATAL,
    LOG_TYPE_ERROR,
    LOG_TYPE_WARN,
    LOG_TYPE_INFO,
    LOG_TYPE_DEBUG
  };

  static const char* TypeToString(const Type& type);

  static bool Initialise(const std::string& fileName, int preserveDays = 30);
  static bool Finalise();

  static bool Fatal(const std::string& message);
  static bool Fatal(const char* format, ...);

  static bool Error(const std::string& message);
  static bool Error(const char* format, ...);

  static bool Warn(const std::string& message);
  static bool Warn(const char* format, ...);

  static bool Info(const std::string& message);
  static bool Info(const char* format, ...);

  static bool Debug(const std::string& message);
  static bool Debug(const char* format, ...);

  static void RolloutHandler(const char* filename,
    std::size_t size,
    el::base::RollingLogFileBasis rollingbasis);
  static void CleanLogs();

  static void copyFile(const char* src, const char* dest);
  static char separator();
  static void removeFile(const char* src);

  std::string GetFileName();
  int GetPreserveDays();

 private:
  bool m_initialised;
  std::string m_fileName;
  int m_preserveDays;

  Log();
  Log(const Log&);
  ~Log();

  static Log& get();

  bool log(const Type& type, const std::string& message);
  bool log(const Type& type, const char* format, va_list& varArgs);

  Log& operator=(const Log&);
};

#endif
