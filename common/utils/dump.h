#if defined(_WIN32)
#ifndef PROJECT_DUMP_H_
#define PROJECT_DUMP_H_

#include <string>

using  std::string;

class DumpService {
 public:
  static void InitMinDump(std::string product_name);
};

#endif //PROJECT_DUMP_H_
#endif




