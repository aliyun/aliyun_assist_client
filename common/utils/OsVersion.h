#ifndef _system_version_h_
#define _system_version_h_

#include <string>
using std::string;

class  OsVersion {
 public:
  static string GetVersion();
  static bool Is64BitOS();
 private:
  static string WindowsGetVersion();
  static string LinuxGetVersion();
  static bool WindowsIs64BitOS();
  static bool LinuxIs64BitOS();
};

#endif
