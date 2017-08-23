/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .cpp
Description: Provide functions to get time value
**************************************************************************/

#include "TimeTool.h"

string Time::GetLocalTime() {

  time_t    rawtime;
  struct tm * timeinfo;
  time(&rawtime);
  timeinfo = localtime(&rawtime);

  char timestr[128] = { 0 };
  strftime(timestr, sizeof(timestr), "%Y%m%d%H%M%S-", timeinfo);
  return string(timestr);
}

int Time::GetDiffTime(time_t start, time_t end) {

  return difftime(end, start);
}

time_t Time::GetCurreTime() {
  time_t ctm;
  time(&ctm);
  return ctm;
}

