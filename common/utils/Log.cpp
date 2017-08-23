#include "Log.h"
#include <ctime>
#include <iostream>

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


bool Log::Initialise( const std::string& fileName ) {
  Log& log = Log::get();

  if( !log.m_initialised ) {
    log.m_fileName = fileName;
    log.m_stream.open( fileName.c_str(),
                       std::ios_base::app | std::ios_base::out );
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
    log.m_stream.close();
    return true;
  }
  return false;
}


void Log::SetThreshold( const Type& type ) {
  Log& log = Log::get();
  log.m_threshold = type;
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


std::string Log::Peek() {
  return Log::get().m_stack.back();
};


bool Log::Push( const std::string& input ) {
  if( !input.empty() ) {
    Debug( input + " BEGIN" );
    Log::get().m_stack.push_back( input );
    return true;
  }
  return false;
}


std::string Log::Pop() {
  Log& log = Log::get();
  if( !log.m_stack.empty() ) {
    std::string temp( log.Peek() );
    log.m_stack.pop_back();
    Debug( temp + " END" );
    return temp;
  }
  return std::string();
}


void Log::PrintStackTrace() {
  Log& log = Log::get();
  std::string temp = "---Stack Trace---\n";

  for( std::vector<std::string>::reverse_iterator i = log.m_stack.rbegin();
       i != log.m_stack.rend(); ++i) {
    temp += "| " + *i + "\n";
  }

  temp += "-----------------";
  log.write( temp.c_str() );
}


Log::Log()
  : m_threshold( LOG_TYPE_INFO ),
    m_fileName(),
    m_stack(),
    m_stream() {
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


void Log::write( const char* format, ... ) {
  char buffer[512];

  va_list varArgs;
  va_start( varArgs, format );
  vsnprintf( buffer, sizeof(buffer), format, varArgs);
  va_end( varArgs );

  //std::cout << buffer << std::endl;
  m_stream  << buffer << std::endl;
}


bool Log::log( const Type& type, const std::string& message ) {
  if( type <= m_threshold ) {
    static const int TIMESTAMP_BUFFER_SIZE = 21;
    char buffer[TIMESTAMP_BUFFER_SIZE];
    time_t timestamp;
    time( &timestamp );
    strftime( buffer, sizeof( buffer ), "%X %x", localtime( &timestamp ) );

    write( "[%s] %s - %s", buffer, TypeToString( type ), message.c_str() );
    return true;
  }
  return false;
}


bool Log::log( const Type& type, const char* format, va_list& varArgs) {
  char buffer[512];
  vsnprintf( buffer, sizeof(buffer), format, varArgs);
  return log( type, buffer );
}


Log& Log::operator=(const Log&) {
  return *this;
}
