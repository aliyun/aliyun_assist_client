#ifndef _XS_SHELL_H_
#define _XS_SHELL_H_

#ifdef __cplusplus
#if __cplusplus
extern "C" {
#endif
#endif  /* __cplusplus */

#include <pthread.h>

#define XS_PATH_CMDSTATEIN   "control/shell/statein"
#define XS_PATH_CMDSTATEOUT  "control/shell/stateout"
#define XS_PATH_CMDSTDIN     "control/shell/stdin"
#define XS_PATH_CMDSTDOUT    "control/shell/stdout"

#define EMPTY_TIMESTAMP "00000000000000:"
#define CMD_MAX_LENGTH 850
#define LENGTH_TIMESTAMP 15
#define BUFFER_SIZE 850

#define STATE_ENABLE    "1"

/*Error message*/
#define ERR_CMD_IS_EMPTY        "550: cmd line is empty\n"
#define ERR_CMD_LAST_IS_RUNNING "551: last cmd is still running\n"
#define ERR_CREATE_PIPE         "552: create pipe error, error"
#define ERR_FORK_PROCESS        "553: fork process error, error"
#define ERR_DUP_STDOUT          "554: dup stdout error, error"
#define ERR_DUP_STDERR          "555: dup stderr error, error"
#define ERR_EXEC_PROCESS        "556: exec process error, error"
#define ERR_WAIT_SUBPROCESS     "557: wait subprocess error, error"
#define ERR_READFILE            "558: read file error, error"
#define ERR_CMD_NOT_SUPPORT     "command is not supported\n"
#define SUC_KICK_VM_CLASSICAL   "\"result\":9, execute kick_vm success\n"
#define SUC_KICK_VM             "\"result\":8, execute kick_vm success\n"

#define SHELL_CMD_TERM_PROCESS  "reset"

typedef enum _CMDStatus{
      CMD_STATUS_RUNNING = 0,
      CMD_STATUS_STOPPED,
}CMDStatus;

typedef void(*XENKICKER)(void);

typedef struct thread_param {
  bool* bTerminated;
  XENKICKER kicker;
} th_param;

extern int XSShellStart(th_param* param,
    pthread_t* pCmdExecThread,
    pthread_t* pCmdReadThread);

#ifdef __cplusplus
#if __cplusplus
}
#endif
#endif  /* __cplusplus */

#endif /*_XS_SHELL_H_*/
