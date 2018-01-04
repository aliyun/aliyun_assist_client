/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .cpp
Description: Provide functions to make process
**************************************************************************/

#ifdef _WIN32
#include <Windows.h>
#else
#include <unistd.h>
#include <linux/limits.h>
#endif // _WIN32

#include "SubProcess.h"
#include "utils/Log.h"
//#include <stdio.h>
//#include <unistd.h>
//#include <signal.h>
SubProcess::SubProcess(string cwd, int time_out) {
  _cwd = cwd;
  _time_out = time_out;
#if defined(_WIN32)
  _hProcess = nullptr;
#else
  pid_ = 0;
#endif
}

SubProcess::SubProcess(string cmd, string cwd) {
  _cmd = cmd;
  _cwd = cwd;
  _time_out = 100;
#if defined(_WIN32)
  _hProcess = nullptr;
#else
  pid_ = 0;
#endif
}

SubProcess::~SubProcess() {}

bool SubProcess::Execute() {
  string out;
  long   exitCode;

  char*  cwd = _cwd.length() ? (char*)_cwd.c_str() : nullptr;
  return ExecuteCmd((char*)_cmd.c_str(),(char*)_cwd.c_str(), false, out, exitCode);
}

bool SubProcess::Execute(string &out, long &exitCode) {
  char*  cwd = _cwd.length() ? (char*)_cwd.c_str() : nullptr;
  return ExecuteCmd((char*)_cmd.c_str(), (char*)_cwd.c_str(), true, out, exitCode);
}


bool SubProcess::ExecuteCmd(char* cmd, const char* cwd, bool isWait, string& out, long &exitCode) {

#ifdef _WIN32
  SECURITY_ATTRIBUTES sattr = { 0 };
  sattr.nLength = sizeof(sattr);
  sattr.bInheritHandle = TRUE;

  HANDLE hChildOutR;
  HANDLE hChildOutW;
  if ( !CreatePipe(&hChildOutR, &hChildOutW, &sattr, 0) ) {
    exitCode = GetLastError();
  }

  SetHandleInformation(hChildOutR, HANDLE_FLAG_INHERIT, 0);
  STARTUPINFOA           si = { 0 };
  PROCESS_INFORMATION    pi = { 0 };

  si.cb = sizeof(si);
  si.hStdOutput = hChildOutW;
  si.hStdError  = hChildOutW;
  si.dwFlags   |= STARTF_USESTDHANDLES;

  EnableWow64(false) ;
  if(strlen(cwd) == 0) {
    cwd = nullptr;
  }

  BOOL ret = CreateProcessA(NULL, cmd, 0, 0, TRUE, 0, 0, cwd, &si, &pi);
  _hProcess = pi.hProcess;
  Log::Info("create process id:%d", GetProcessId(_hProcess));
  EnableWow64(true);

  if ( !ret ) {
    CloseHandle(hChildOutR);
    CloseHandle(hChildOutW);
    return false;
  }
  string task_out;
  for (int i = 0; i < 2 && isWait;) {
    DWORD  len = 0;
    while ( PeekNamedPipe(hChildOutR, 0, 0, 0, &len, 0) && len) {
      CHAR  output[0x1000] = { 0 };
      ReadFile(hChildOutR, output, sizeof(output) - 1, &len, 0);
      task_out = task_out + output;
    };

    if ( WAIT_OBJECT_0 ==
         WaitForSingleObject(pi.hProcess, INFINITE) ) {
      i++;
      DWORD exitCodeD;
      GetExitCodeProcess(pi.hProcess, &exitCodeD);
      exitCode = exitCodeD;
    }
  }
  out = task_out;
  CloseHandle(hChildOutR);
  CloseHandle(hChildOutW);
  CloseHandle(pi.hProcess);
  CloseHandle(pi.hThread);
  return true;

#else
  return ExecuteCMD_LINUX(cmd, cwd, isWait, out, exitCode);

#endif
}

bool SubProcess::RunModule(string moduleName) {
#ifdef _WIN32
  STARTUPINFOA si;
  PROCESS_INFORMATION pi;

  ZeroMemory(&si, sizeof(si));
  si.cb = sizeof(si);
  ZeroMemory(&pi, sizeof(pi));

  CHAR Buffer[MAX_PATH];
  DWORD dwRet = GetModuleFileNameA(NULL, Buffer, MAX_PATH);

  if (dwRet == 0 || dwRet > MAX_PATH) {
    Log::Error("get module file name failed,error code is %d", GetLastError());
    return FALSE;
  }

  string filePath = Buffer;
  filePath = filePath.substr(0, filePath.find_last_of('\\',
    filePath.length()) + 1);
  filePath = filePath + moduleName + " ";

  string command_line = filePath + _cmd;

  if (!CreateProcessA(nullptr,   // No module name (use command line)
    (LPSTR)command_line.c_str(),        // Command line
    NULL,           // Process handle not inheritable
    NULL,           // Thread handle not inheritable
    FALSE,          // Set handle inheritance to FALSE
    0,              // No creation flags
    NULL,           // Use parent's environment block
    NULL,           // Use parent's starting directory
    &si,            // Pointer to STARTUPINFO structure
    &pi)           // Pointer to PROCESS_INFORMATION structure
    ) {
    Log::Error("createProcess failed,error code is %d", GetLastError());
    return FALSE;
  }

  // Wait until child process exits.
  DWORD ret = WaitForSingleObject(pi.hProcess, 10 * 1000);

  // Close process and thread handles.
  CloseHandle(pi.hProcess);
  CloseHandle(pi.hThread);

  // If the object is not sigalled, we think the call is failure.
  if (ret != WAIT_OBJECT_0) {
    Log::Warn("process is not completed correctly,error code is %d",
      GetLastError());
    return false;
  }

  return true;
#else
  char buffer[PATH_MAX];
  getcwd(buffer, PATH_MAX);
  string filePath = buffer;
  string command_line = filePath + "/" + moduleName + " " + _cmd;
  FILE *ptr;
  if ((ptr = popen(command_line.c_str(), "r")) != NULL) {
    pclose(ptr);
    ptr = NULL;
    return true;
  }
  else {
    return false;
  }
#endif
}

bool SubProcess::IsExecutorExist(string guid) {

#ifdef _WIN32
  HANDLE hMutexInstance = CreateMutexA(NULL, FALSE, guid.c_str());  //创建互斥
  if (NULL == hMutexInstance) {
    return false;
  }
  if (GetLastError() == ERROR_ALREADY_EXISTS) {
    OutputDebugStringA("IsExecutorExist return");
    return true;
  }
  return false;

#else
//    char cmd[128] = {0};
//    sprintf(cmd, "ps -ef|grep %s ",guid);

//    ExecuteCMD_LINUX(char* cmd, const char* cwd, bool isWait, string& out, long &exitCode);
//    FILE *pstr; ,buff[512],*p;
//    pid_t pID;
//    int pidnum;
//    char *name= "ping ";//要查找的进程名
//    int ret=3;

//    pstr=popen(cmd, "r");//

//    if(pstr==NULL)
//    { return 1; }
//    memset(buff,0,sizeof(buff));
//    fgets(buff,512,pstr);
//    p=strtok(buff, " ");
//    p=strtok(NULL, " "); //这句是否去掉，取决于当前系统中ps后，进程ID号是否是第一个字段 pclose(pstr);
//    if(p==NULL)
//    { return 1; }
//    if(strlen(p)==0)
//    { return 1; }
//    if((pidnum=atoi(p))==0)
//    { return 1; }
//    printf("pidnum: %d\n",pidnum);
//    pID=(pid_t)pidnum;
//    ret=kill(pID,0);//这里不是要杀死进程，而是验证一下进程是否真的存在，返回0表示真的存在
//    printf("ret= %d \n",ret);
//    if(0==ret)
//        printf("process: %s exist!\n",name);
//    else printf("process: %s not exist!\n",name);

  return false;

#endif
}

#ifndef _WIN32

#include    <sys/wait.h>  
#include    <errno.h>  
#include    <fcntl.h>  

//#include    "ourhdr.h"  
  
static pid_t    *childpid = NULL;  
                        /* ptr to array allocated at run-time */  
static int      maxfd;  /* from our open_max(), {Prog openmax} */  
  
#define SHELL   "/bin/sh"  

FILE * SubProcess::popen2(const char *cmdstring, const char *type, const char *cwd)  
{  
    int     i, pfd[2];  
    pid_t   pid;  
    FILE    *fp;  
  
            /* only allow "r" or "w" */  
    if ((type[0] != 'r' && type[0] != 'w') || type[1] != 0) {  
        errno = EINVAL;     /* required by POSIX.2 */  
        return(NULL);  
    }  
  
    if (childpid == NULL) {     /* first time through */  
                /* allocate zeroed out array for child pids */  
        maxfd = 1024 * 8;  
        if ( (childpid = (pid_t *)calloc(maxfd, sizeof(pid_t))) == NULL)  
            return(NULL);  
    }  
  
    if (pipe(pfd) < 0)  
        return(NULL);   /* errno set by pipe() */  
  
    if ( (pid = fork()) < 0)  
        return(NULL);   /* errno set by fork() */  
    else if (pid == 0) {                            /* child */  
        chdir(cwd); //change dir

        if (*type == 'r') {  
            close(pfd[0]);  
            if (pfd[1] != STDOUT_FILENO) {  
                dup2(pfd[1], STDOUT_FILENO);  
                close(pfd[1]);  
            }  
        } else {  
            close(pfd[1]);  
            if (pfd[0] != STDIN_FILENO) {  
                dup2(pfd[0], STDIN_FILENO);  
                close(pfd[0]);  
            }  
        }  
            /* close all descriptors in childpid[] */  
        for (i = 0; i < maxfd; i++)  
            if (childpid[ i ] > 0)  
                close(i);  
  
        execl(SHELL, "sh", "-c", cmdstring, (char *) 0);  
        _exit(127);  
    }  
                                /* parent */  
    pid_ = pid;
    if (*type == 'r') {  
        close(pfd[1]);  
        if ( (fp = fdopen(pfd[0], type)) == NULL)  
            return(NULL);  
    } else {  
        close(pfd[0]);  
        if ( (fp = fdopen(pfd[1], type)) == NULL)  
            return(NULL);  
    }  
    Log::Info("pid =%d", pid);
    childpid[fileno(fp)] = pid; /* remember child pid for this fd */  
    return(fp);  
}  

int SubProcess::pclose2(FILE *fp)  
{  
    int     fd, stat;  
    pid_t   pid;  
  
    if (childpid == NULL)  
        return(-1);     /* popen() has never been called */  
  
    fd = fileno(fp);  
    if ( (pid = childpid[fd]) == 0)  
        return(-1);     /* fp wasn't opened by popen() */  
  
    childpid[fd] = 0;  
    if (fclose(fp) == EOF)  
        return(-1);  
  
    while (waitpid(pid, &stat, 0) < 0)  
        if (errno != EINTR)  
            return(-1); /* error other than EINTR from waitpid() */  
  
    return(stat);   /* return child's termination status */  
} 

bool SubProcess::ExecuteCMD_LINUX(char* cmd, const char* cwd, bool isWait, string& out, long &exitCode) {
  char tmp_buf[1024] = {0};
  char result[1024 * 10] = {0};
  FILE* ptr = nullptr;
  if ((ptr = popen2(cmd, "r", cwd)) != NULL) {
    while (fgets(tmp_buf, 1024, ptr) != NULL) {
      strcat(result, tmp_buf);
      if (strlen(result)>1024*8) break;
    }
    Log::Info("result:%s", result);
    out = result;
    exitCode = 0;
    pclose2(ptr);
    ptr = NULL;
    return true;
  } else  {
    Log::Error("popen failed.");
    out = "";
    exitCode = -1;
    return false;
  }
}
#endif

#if defined(_WIN32)
HANDLE SubProcess::get_id() {
  return _hProcess;
}
#else

pid_t SubProcess::get_id() {
  return pid_;
}
#endif

#ifdef _WIN32
void SubProcess::EnableWow64(bool enable) {

  typedef BOOL(APIENTRY *_Wow64EnableWow64FsRedirection)(BOOL);
  _Wow64EnableWow64FsRedirection  fun;
  HMODULE hmod = GetModuleHandleA("Kernel32.dll");

  fun = (_Wow64EnableWow64FsRedirection)
        GetProcAddress(hmod, "Wow64EnableWow64FsRedirection");

  if (fun) fun(enable);
};

#endif
