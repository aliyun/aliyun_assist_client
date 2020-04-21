/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .h
Description: Provide functions to get time value
**************************************************************************/

#ifndef __TIMETOOL_H_
#define __TIMETOOL_H_

#include <time.h>
#include <string>

using  std::string;

class TimeTool {
 public:
  static string GetLocalTime();
  static int GetDiffTime(time_t start, time_t end);
  static int64_t GetAccurateTime();
};

#endif //__TIMETOOL_H_




