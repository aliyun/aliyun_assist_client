#ifndef COMMON_UTILS_VERSIONCOMPARATOR_H_
#define COMMON_UTILS_VERSIONCOMPARATOR_H_

#include <string>

class  VersionComparator {
 public:
  static int CompareVersions(const std::string& a, const std::string& b);
};

#endif  // COMMON_UTILS_VERSIONCOMPARATOR_H_
