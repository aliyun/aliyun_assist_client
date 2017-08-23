// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#ifndef SERVICE_XS_SHELL_H_
#define SERVICE_XS_SHELL_H_

#ifdef __cplusplus
#if __cplusplus
extern "C" {
#endif
#endif  /* __cplusplus */

#include <windows.h>
#include <stdio.h>
#include <process.h>
#include <string.h>
#include <tlhelp32.h>
#include <strsafe.h>
#include <stdlib.h>
#define XS_PATH_CMDSTATEIN   "control/shell/statein"
#define XS_PATH_CMDSTATEOUT  "control/shell/stateout"
#define XS_PATH_CMDSTDIN     "control/shell/stdin"
#define XS_PATH_CMDSTDOUT    "control/shell/stdout"

#define LENGTH_TIMESTAMP 15
#define EMPTY_TIMESTAMP "00000000000000:"
#define CMD_MAX_LENGTH 850
#define BUFFER_SIZE 850

#define STATE_ENABLE    "1"

/*Error message*/
#define ERR_CMD_IS_EMPTY        "51: cmd line is empty\n"
#define ERR_CMD_LAST_IS_RUNNING "52: last cmd is still running\n"
#define ERR_CREATE_PIPE         "53: create pipe error, last error"
#define ERR_CREATE_PROCESS      "54: create process error, last error"
#define ERR_READFILE            "55: read file error, last error"
#define ERR_CMD_NOT_SUPPORT     "command is not supported"
#define SUC_KICK_VM             "kick vm success"

#define SHELL_CMD_TERM_PROCESS  "reset"

typedef enum _CMDStatusc {
  CMD_STATUS_RUNNING = 0,
  CMD_STATUS_STOPPED,
}CMDStatus;

typedef void(*XENKICKER)(void);

typedef struct thread_param {
  BOOL* terminatingService;
  XENKICKER kicker;
} th_param;

extern int XSShellStart(th_param* param,
    HANDLE& hCmdExecThread,
    HANDLE& hCmdReadThread);

#ifdef __cplusplus
#if __cplusplus
}
#endif
#endif  /* __cplusplus */

#endif  // SERVICE_XS_SHELL_H_
