#ifndef LOG_H
#define LOG_H

#include <string.h>
#include <vector>
#include <stdio.h>
#include <iostream>
#include <fstream>
#include <stdarg.h>


#define PUSH_LOG_STACK \
	const StackLogger stackLoggerTempObject = StackLogger( __FUNCTION__ );


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

  static bool Initialise(const std::string& fileName);
  static bool Finalise();

  static void SetThreshold(const Type& type);

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

  static std::string Peek();
  static bool Push(const std::string& input);
  static std::string Pop();
  static void PrintStackTrace();

 private:
  Type m_threshold;
  bool m_initialised;
  std::string m_fileName;
  std::vector<std::string> m_stack;
  std::ofstream m_stream;

  Log();
  Log(const Log&);
  ~Log();

  static Log& get();

  void write(const char* format, ...);

  bool log(const Type& type, const std::string& message);
  bool log(const Type& type, const char* format, va_list& varArgs);

  Log& operator=(const Log&);
};

struct StackLogger {
  StackLogger(const std::string& input) {
    Log::Push( input );
  }

  ~StackLogger() {
    Log::Pop();
  }
};

#endif
