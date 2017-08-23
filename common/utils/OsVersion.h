#ifndef _system_version_h_
#define _system_version_h_

#include <string>
using std::string;

class  OsVersion {
 public:
  static string GetVersion();
 private:
  static string WindowsGetVersion();
  static string LinuxGetVersion();
};

#endif
