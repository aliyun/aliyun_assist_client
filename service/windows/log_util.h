// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#ifndef SERVICE_LOG_UTIL_H_
#define SERVICE_LOG_UTIL_H_

#include <windows.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <strsafe.h>

#define LOG_PATH "shutdownmon.log"

extern int log_init(void);
extern void log_close(void);
extern int log2local(char * format, ...);

#endif  // SERVICE_LOG_UTIL_H_
