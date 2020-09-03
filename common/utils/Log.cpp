#include "Log.h"
#include <stdio.h>
#include <ctime>
#include "utils/DirIterator.h"
#if defined (_WIN32)
#include <windows.h>
#else
#include <unistd.h>
#include <sys/time.h>
#endif

#if defined(_WIN32)
#define MAX_FIlE_PATH   260
#else
#define MAX_FIlE_PATH   1024
#endif


INITIALIZE_EASYLOGGINGPP

#if !defined(_WIN32)
#include <signal.h>
#include "utils/backtrace.h"

static void unwindingCrashHandler(int sig) {
    // Restore default signal handler
#if defined(ELPP_HANDLE_SIGABRT)
    int i = 0;  // SIGABRT is at base::consts::kCrashSignals[0]
#else
    int i = 1;
#endif  // defined(ELPP_HANDLE_SIGABRT)
    for (; i < el::base::consts::kCrashSignalsCount; ++i) {
        signal(el::base::consts::kCrashSignals[i].numb, SIG_DFL);
    }
    el::Helpers::logCrashReason(sig, false);
    LOG(FATAL) << "======= Backtrace: =========" << std::endl << StackUnwind();
    el::Helpers::crashAbort(sig);
}
#endif

const char* Log::TypeToString( const Type& type ) {
  switch( type ) {
  case LOG_TYPE_FATAL:
    return "FATAL";
  case LOG_TYPE_ERROR:
    return "ERROR";
  case LOG_TYPE_WARN:
    return "WARN ";
  case LOG_TYPE_INFO:
    return "INFO ";
  case LOG_TYPE_DEBUG:
    return "DEBUG";
  default:
    break;
  }
  return "UNKWN";
}


bool Log::Initialise(const std::string& fileName, int preserveDays) {
    Log& log = Log::get();

  if( !log.m_initialised ) {
    log.m_fileName = fileName;
    log.m_preserveDays = preserveDays;

    el::Loggers::addFlag(el::LoggingFlag::DisableApplicationAbortOnFatalLog);
    el::Loggers::addFlag(el::LoggingFlag::StrictLogFileTimeCheck);
    el::Loggers::reconfigureAllLoggers(el::ConfigurationType::ToStandardOutput, "false");
    el::Loggers::reconfigureAllLoggers(el::ConfigurationType::Filename, log.m_fileName);
    el::Loggers::reconfigureAllLoggers(el::ConfigurationType::LogFileRollingTime, "day");
    el::Helpers::installPreRollOutCallback(RolloutHandler);
#if !defined(_WIN32)
    el::Helpers::setCrashHandler(unwindingCrashHandler);
#endif

    log.m_initialised = true;
    Info( "LOG INITIALISED" );
    return true;
  }
  return false;
}


bool Log::Finalise() {
  Log& log = Log::get();

  if( log.m_initialised ) {
    Info( "LOG FINALISED" );
    el::Helpers::uninstallPreRollOutCallback();
    return true;
  }
  return false;
}


bool Log::Fatal( const std::string& message ) {
  return Log::get().log( LOG_TYPE_FATAL, message );
}


bool Log::Fatal( const char* format, ... ) {
  va_list varArgs;
  va_start( varArgs, format );
  bool success = Log::get().log( LOG_TYPE_FATAL, format, varArgs);
  va_end( varArgs );
  return success;
}

bool Log::Error( const std::string& message ) {
  return Log::get().log( LOG_TYPE_ERROR, message );
}


bool Log::Error( const char* format, ... ) {
  va_list varArgs;
  va_start( varArgs, format );
  bool success = Log::get().log( LOG_TYPE_ERROR, format, varArgs);
  va_end( varArgs );
  return success;
}


bool Log::Warn( const std::string& message ) {
  return Log::get().log( LOG_TYPE_WARN, message );
}


bool Log::Warn( const char* format, ... ) {
  va_list varArgs;
  va_start( varArgs, format );
  bool success = Log::get().log( LOG_TYPE_WARN, format, varArgs);
  va_end( varArgs );
  return success;
}


bool Log::Info( const std::string& message ) {
  return Log::get().log( LOG_TYPE_INFO, message );
}


bool Log::Info( const char* format, ... ) {
  va_list varArgs;
  va_start( varArgs, format );
  bool success = Log::get().log( LOG_TYPE_INFO, format, varArgs);
  va_end( varArgs );
  return success;
}


bool Log::Debug( const std::string& message ) {
  return Log::get().log( LOG_TYPE_DEBUG, message );
}


bool Log::Debug( const char* format, ... ) {
  va_list varArgs;
  va_start( varArgs, format );
  bool success = Log::get().log( LOG_TYPE_DEBUG, format, varArgs);
  va_end( varArgs );
  return success;
}

void Log::RolloutHandler(const char* filename, std::size_t size, el::base::RollingLogFileBasis rollingbasis)
{
  switch (rollingbasis)
  {
  case el::base::RollingLogFileBasis::RollLog_FileSize:
    /// ����С������־�ļ�
    break;
  case el::base::RollingLogFileBasis::RollLog_DateTime:
    /// ��ʱ�������־�ļ�
  {
    time_t currenttime = time(NULL);
    currenttime -= 24 * 3600;

    struct::tm oneDayAgo;
#if defined(_WIN32)
    localtime_s(&oneDayAgo, &currenttime);
#else
    localtime_r(&currenttime, &oneDayAgo);
#endif

    std::string filenameTemp = filename;
    int pos = filenameTemp.rfind('.');
    filenameTemp = filenameTemp.substr(0, pos);
    char backupFile[MAX_FIlE_PATH] = { 0 };
#if defined(_WIN32)
    sprintf_s(backupFile, MAX_FIlE_PATH, 
      "%s_%04d%02d%02d%02d%02d.log",
      filenameTemp.c_str(),
      oneDayAgo.tm_year + 1900,
      oneDayAgo.tm_mon + 1,
      oneDayAgo.tm_mday,
      oneDayAgo.tm_hour,
      oneDayAgo.tm_min);
#else
    snprintf(backupFile, MAX_FIlE_PATH,
      "%s_%04d%02d%02d%02d%02d.log",
      filenameTemp.c_str(),
      oneDayAgo.tm_year + 1900,
      oneDayAgo.tm_mon + 1,
      oneDayAgo.tm_mday,
      oneDayAgo.tm_hour,
      oneDayAgo.tm_min);
#endif

    /// �Զ�����־����
    Log::copyFile(filename, backupFile);
  }
  break;
  default:
    break;
  }

  Log::CleanLogs();
}

void Log::CleanLogs() {
  int preserve_days = Log::get().GetPreserveDays();
  std::string filenameTemp = Log::get().GetFileName();
  int pos = filenameTemp.rfind(Log::separator());
  std::string dir = filenameTemp.substr(0, pos);

  if (dir.empty()) {
    return;
  }

  DirIterator it_dir(dir.c_str());
  while (it_dir.next()) {
    std::string name = it_dir.fileName();
    if (name == "." || name == "..")
      continue;

    if (it_dir.isDir())
      continue;

    std::string file_path = dir + Log::separator() + name;
    if (file_path == filenameTemp)
      continue;

    time_t currenttime = time(NULL);
    struct stat file_info;
    stat(file_path.c_str(), &file_info);
    double totalT = difftime(currenttime, file_info.st_ctime);
    if (totalT > preserve_days * 24 * 3600) {
      Log::removeFile(file_path.c_str());
    }
  }

  return;
}

void Log::copyFile(const char* src, const char* dest) {
#if !defined _WIN32
  std::ifstream inputFile(src, std::ios::binary);
  std::ofstream outputFile(dest, std::ios::binary | std::ios::trunc);


  outputFile << inputFile.rdbuf();

#else
  CopyFileA(src, dest, FALSE);
#endif
}

char Log::separator() {
#ifdef _WIN32
  return '\\';
#else
  return '/';
#endif
}

void Log::removeFile(const char* src) {
#if !defined _WIN32
  unlink(src);
#else
  DeleteFileA(src);
#endif
}


std::string Log::GetFileName() {
  return m_fileName;
}

int Log::GetPreserveDays() {
  return m_preserveDays > 0 ? m_preserveDays : 0;
}

Log::Log()
  : m_fileName() {
}


Log::Log(const Log&) {
}


Log::~Log() {
  Finalise();
}


Log& Log::get() {
  static Log log;
  return log;
}

bool Log::log( const Type& type, const std::string& message ) {
  switch (type) {
  case LOG_TYPE_FATAL:
    LOG(FATAL) << message;
    return true;
  case LOG_TYPE_ERROR:
    LOG(ERROR) << message;
    return true;
  case LOG_TYPE_WARN:
    LOG(WARNING) << message;
    return true;
  case LOG_TYPE_INFO:
    LOG(INFO) << message;
    return true;
  case LOG_TYPE_DEBUG:
    LOG(DEBUG) << message;
    return true;
  default:
    break;
  }
  return false;
}


bool Log::log( const Type& type, const char* format, va_list& varArgs) {
  char buffer[1024*256];
  vsnprintf( buffer, sizeof(buffer), format, varArgs);
  return log( type, buffer );
}


Log& Log::operator=(const Log&) {
  return *this;
}

