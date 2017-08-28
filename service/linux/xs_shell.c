#include <stdio.h>
#include <string.h>
#include <stdlib.h> 
#include <unistd.h>
#include <signal.h>
#include <pthread.h>
#include <stdarg.h>
#include <errno.h>

#include <xs.h>
#include <xs_shell.h>
#include "utils/Log.h"

pthread_mutex_t  gmutex_stdin = PTHREAD_MUTEX_INITIALIZER;
pthread_cond_t   gcond_stdin  = PTHREAD_COND_INITIALIZER;

CMDStatus gcmd_status = CMD_STATUS_STOPPED;
char cmdbuf[CMD_MAX_LENGTH+LENGTH_TIMESTAMP+1];
pid_t g_childpid = 0;

bool write_xenstore(struct xs_handle *h, xs_transaction_t t, const char *path, const void *data, unsigned int len, const char *ptimestamp) {
    char writebuf[BUFFER_SIZE + LENGTH_TIMESTAMP];
    int str_len;

    if (ptimestamp != NULL) {
        if (strlen(ptimestamp) >= LENGTH_TIMESTAMP)
            memcpy(writebuf, ptimestamp, LENGTH_TIMESTAMP);
        memcpy(writebuf + LENGTH_TIMESTAMP, data, len);
        str_len = len + LENGTH_TIMESTAMP;
    } else {
        memcpy(writebuf, data, len);
        str_len = len;
    }

    Log::Info("xs_write: [%s] [%.*s] [%d]", path, str_len, writebuf, str_len);
    return xs_write(h, t, path, writebuf, str_len);
}

void exec_cmd(struct xs_handle *xsh, XENKICKER kicker) {
    char *ptimestamp = EMPTY_TIMESTAMP;

    //check
    Log::Info("cmdbuf = %s", cmdbuf);

    if (strlen(cmdbuf) <= LENGTH_TIMESTAMP) {
        write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTDOUT, ERR_CMD_IS_EMPTY, strlen(ERR_CMD_IS_EMPTY), ptimestamp);
        return;
    }

    ptimestamp = cmdbuf;
    
    char* strCmd = strstr(cmdbuf, "kick_vm");
    if ((strCmd != NULL) && !strcmp(strCmd, "kick_vm")) {
        kicker();
        write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTDOUT, SUC_KICK_VM, strlen(SUC_KICK_VM), ptimestamp);
    } else {
        write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTDOUT, ERR_CMD_NOT_SUPPORT, strlen(ERR_CMD_NOT_SUPPORT), ptimestamp);
    }
}

static void* shell_cmd_check_thread(void *arg) {
    struct xs_handle *watch_xsh;
    struct xs_handle *xsh;
    char **res;
    char* token = "0";
    int num;

    Log::Info("check thread start");

    if((watch_xsh = xs_domain_open()) == NULL) {
        Log::Error("Connect to xenbus failed: %s", strerror(errno));
        return (void*)-1;
    }
    xs_watch(watch_xsh, XS_PATH_CMDSTATEIN, token);

    if((xsh = xs_domain_open()) == NULL) {
        Log::Error("Connect to xenbus failed: %s", strerror(errno));
        return (void*)-1;
    }
    write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTATEOUT, STATE_ENABLE, strlen(STATE_ENABLE), NULL);

    for(;;) {
        if((res = xs_read_watch(watch_xsh, &num)) == NULL)
            continue;

        write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTATEOUT, STATE_ENABLE, strlen(STATE_ENABLE), NULL);

        free(res);
    }

    Log::Info("check thread end");
    return (void*)0;
}

static void* shell_cmd_read_thread(void *arg) {
    pthread_t tshell_cmd_check;
    struct xs_handle *watch_xsh;
    struct xs_handle *xsh;
    char **res;
    char* token = "0";
    int num;
    char* buf;
    unsigned int len;
    unsigned int cmdlen;
    int ret;

    Log::Info("read thread start");

    bool* bTerminated = (bool*)arg;
    
    if((watch_xsh = xs_domain_open()) == NULL) {
        Log::Error("Connect to xenbus failed: %s", strerror(errno));
        return (void*)-1;
    }
    xs_watch(watch_xsh, XS_PATH_CMDSTDIN, token);

    if((ret = pthread_create(&tshell_cmd_check, NULL, shell_cmd_check_thread, NULL)) != 0) {
        Log::Error("shell_cmd_check_thread create failed: %s", strerror(errno));
        return (void*)-1;
    }

    if((xsh = xs_domain_open()) == NULL) {
        Log::Error("Connect to xenbus failed: %s", strerror(errno));
        return (void*)-1;
    }

    while(!(*bTerminated)) {
        if((res = xs_read_watch(watch_xsh, &num)) == NULL)
            continue;

        buf = xs_read(xsh, XBT_NULL, XS_PATH_CMDSTDIN, &len);

        if (buf == NULL) {
            free(res);
            continue;
        }

        Log::Info("new event: %s", buf);

        if(memcmp(buf, SHELL_CMD_TERM_PROCESS, strlen(SHELL_CMD_TERM_PROCESS)) == 0) {
            if(g_childpid)
                kill(g_childpid, 9);
            goto cont;
        }

        pthread_mutex_lock(&gmutex_stdin);

        if(gcmd_status == CMD_STATUS_RUNNING) {
            write_xenstore(xsh, XBT_NULL, XS_PATH_CMDSTDOUT, ERR_CMD_LAST_IS_RUNNING, strlen(ERR_CMD_LAST_IS_RUNNING), buf);
            pthread_mutex_unlock(&gmutex_stdin);
            goto cont;
        }

        gcmd_status = CMD_STATUS_RUNNING;
        cmdlen = len < (CMD_MAX_LENGTH + LENGTH_TIMESTAMP) ? len : (CMD_MAX_LENGTH + LENGTH_TIMESTAMP);
        memcpy(cmdbuf, buf, cmdlen);
        cmdbuf[cmdlen] = '\0';

        pthread_cond_signal(&gcond_stdin);
        pthread_mutex_unlock(&gmutex_stdin);
cont:
        if(buf != NULL)
            free(buf);
        free(res);
    }

    Log::Info("read thread end");
    return (void*)0;
}

static void* shell_cmd_exec_thread(void *arg) {
    struct xs_handle *xsh;     

    Log::Info("exec thread start");
    th_param *pargs;
    pargs = (th_param*)arg;
    if((xsh = xs_domain_open()) == NULL) {
        Log::Error("Connect to xenbus failed: %s", strerror(errno));
        return (void*)-1;
    }

    while(!(*pargs->bTerminated)) {
        /*wait signal from shell_cmd_read_thread*/
        pthread_mutex_lock(&gmutex_stdin);
        while (gcmd_status != CMD_STATUS_RUNNING)
            pthread_cond_wait(&gcond_stdin, &gmutex_stdin);
        pthread_mutex_unlock(&gmutex_stdin);

        Log::Info("exec start");
        exec_cmd(xsh, pargs->kicker);
        Log::Info("exec end");

        pthread_mutex_lock(&gmutex_stdin);
        gcmd_status = CMD_STATUS_STOPPED;
        pthread_mutex_unlock(&gmutex_stdin);
    }

    Log::Info("exec thread end");
    return (void*)0;
}

int XSShellStart(th_param* param,
    pthread_t* pCmdExecThread,
    pthread_t* pCmdReadThread) {

    Log::Info("gshell start");

    if(pthread_create(pCmdExecThread, NULL, shell_cmd_exec_thread, param) != 0) {
        Log::Error("shell_cmd_exec_thread create failed: %s", strerror(errno));
        return 0;
    }
    if(pthread_create(pCmdReadThread, NULL, shell_cmd_read_thread, param->bTerminated) != 0) {
        Log::Error("shell_cmd_read_thread create failed: %s", strerror(errno));
        return 0;
    }  

    Log::Info("Threads created");
    return 1;
}

