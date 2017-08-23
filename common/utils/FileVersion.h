#ifndef _file_version_h_
#define _file_version_h_

#include <string>
using std::string;

class  FileVersion {
 public:
  static string GetFileVersion();
 private:
#if defined(_WIN32)
  static string WindowsGetFileVersion();
#else
  static string LinuxGetFileVersion();
#endif
};

#endif
