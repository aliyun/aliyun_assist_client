/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .h
Description: Provide functions to get time value
**************************************************************************/

#ifndef PROJECT_TIME_H_
#define PROJECT_TIME_H_

#include <time.h>
#include <string>

using  std::string;

class Time {
 public:
  static string GetLocalTime();
  static int GetDiffTime(time_t start, time_t end);
  static time_t GetCurreTime();

};

#endif //PROJECT_TIME_H_




