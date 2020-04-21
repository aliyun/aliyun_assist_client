#include "TimeTool.h"
#ifdef _WIN32
#include <windows.h>
#else
#include <sys/time.h>
#endif // _WIN32

#ifdef _WIN32
int gettimeofday(struct timeval *tp, void *tzp)
{
  time_t clock;
  struct tm tm;
  SYSTEMTIME wtm;
  GetLocalTime(&wtm);
  tm.tm_year = wtm.wYear - 1900;
  tm.tm_mon = wtm.wMonth - 1;
  tm.tm_mday = wtm.wDay;
  tm.tm_hour = wtm.wHour;
  tm.tm_min = wtm.wMinute;
  tm.tm_sec = wtm.wSecond;
  tm.tm_isdst = -1;
  clock = mktime(&tm);
  tp->tv_sec = clock;
  tp->tv_usec = wtm.wMilliseconds * 1000;
  return (0);
}
#endif // _WIN32

string TimeTool::GetLocalTime() {
  time_t    rawtime;
  struct tm * timeinfo;
  time(&rawtime);
  timeinfo = localtime(&rawtime);

  char timestr[128] = { 0 };
  strftime(timestr, sizeof(timestr), "%Y-%m-%d %H:%M:%S", timeinfo);
  return string(timestr);
}

int TimeTool::GetDiffTime(time_t start, time_t end) {
  return difftime(end, start);
}

int64_t TimeTool::GetAccurateTime() {
  struct timeval tv;
  gettimeofday(&tv, NULL);
  int64_t sec = tv.tv_sec;
  return sec * 1000 + tv.tv_usec / 1000;
}

