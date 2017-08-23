// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#pragma warning(disable: 4201)
#include "./log_util.h"

FILE *log_file = NULL;
HANDLE gMutexLog = NULL;

int log_init(void) {
  if ((log_file = fopen(LOG_PATH, "a")) == NULL)
    return -1;
  gMutexLog = CreateMutex(NULL, FALSE, NULL);
  if (gMutexLog == NULL)
    return -1;
  return 0;
}

void log_close(void) {
  if (log_file)
    fclose(log_file);
  if (gMutexLog)
    CloseHandle(gMutexLog);
}

int log2local(char * format, ...) {
  va_list argptr;
  SYSTEMTIME ti;

  if (gMutexLog == NULL)
    return -1;
  if (log_file == NULL)
    return -1;

  WaitForSingleObject(gMutexLog, INFINITE);

  GetLocalTime(&ti);
  fprintf(log_file, "[%04d%02d%02d,%02d:%02d:%02d] [%d] ",
      ti.wYear, ti.wMonth, ti.wDay, ti.wHour, ti.wMinute,
      ti.wSecond, GetCurrentThreadId());
  va_start(argptr, format);
  vfprintf(log_file, format, argptr);
  va_end(argptr);
  fflush(log_file);

  ReleaseMutex(gMutexLog);

  return 0;
}
