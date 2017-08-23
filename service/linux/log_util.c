#include <pthread.h>
#include <stdio.h>
#include <stdarg.h>
#include <time.h>

#include <log_util.h>

FILE *log_file;

pthread_mutex_t  gmutex_log = PTHREAD_MUTEX_INITIALIZER;

int log_init(void) {
    if ( (log_file = fopen(LOG_PATH, "a+")) == NULL) return -1;
    return 0;
}

void log_close(void) {
    if (log_file)
        fclose(log_file);
}

int log2local(const char * format, ... ) {
    va_list argptr;
    int cnt;
    struct tm *ptm;
    long ts;

    if (log_file == NULL)
        return -1;

    pthread_mutex_lock(&gmutex_log);

    ts = time(NULL);
    ptm = localtime(&ts);
    fprintf(log_file, "[%04d%02d%02d,%02d:%02d:%02d] [%d] ", ptm->tm_year+1900, ptm->tm_mon+1, ptm->tm_mday, ptm->tm_hour, ptm->tm_min, ptm->tm_sec, pthread_self());

    va_start(argptr, format);
    cnt = vfprintf(log_file, format, argptr);
    va_end(argptr);
    fflush(log_file);

    pthread_mutex_unlock(&gmutex_log);
    return(cnt);
}
