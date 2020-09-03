#ifndef COMMON_UTILS_SYSTEMINFO_H_
#define COMMON_UTILS_SYSTEMINFO_H_

#include <string>

class  SystemInfo {
 public:
  static std::string GetAllIPs();
#ifdef _WIN32
  static unsigned long GetWindowsDefaultLang();
#endif
 private:
#ifdef _WIN32
  static bool InitWSA();
  static void ReleaseWSA();
#endif // _WIN32
};

#endif  // COMMON_UTILS_SYSTEMINFO_H_
