#ifndef _LOG_UTIL_H_
#define _LOG_UTIL_H_

#define LOG_PATH "/var/log/gshell.log"

int log_init();
void log_close();
/*
 * log to local file
 */
int log2local(const char * format, ... );

#endif /*_LOG_UTIL_H_*/
