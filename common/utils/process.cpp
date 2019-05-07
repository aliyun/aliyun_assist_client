#include "process.h"
#include <memory.h>
#include <memory>
#include <time.h>
#include "utils/Log.h"
#ifdef _WIN32
#include <windows.h>
#pragma comment(lib, "rpcrt4.lib") 
#else
#include <sys/wait.h>
#include <cstdlib>
#include <unistd.h>
#include <signal.h>
#include <stdexcept>
#endif // _WIN32

#ifndef max
#define max(a,b) ((a) > (b) ? (a) : (b))
#endif

Process::Process(string command, string path) {
	m_command = command;
	m_path = path;
};


#define BUFSIZE 1024
#ifdef _WIN32

struct Handle {
public:
	Handle(): m_handle(INVALID_HANDLE_VALUE) { }
	Handle(HANDLE handle):m_handle(handle) { }

	~Handle() {
		close();
	}
	void close(){
		if (m_handle != INVALID_HANDLE_VALUE)
			CloseHandle(m_handle);
	}
	HANDLE detach() {
		HANDLE old_handle = m_handle;
		m_handle = INVALID_HANDLE_VALUE;
		return old_handle;
	}
	operator HANDLE() const noexcept { return m_handle; }
	HANDLE* operator&() noexcept { return &m_handle; }
private:
	HANDLE m_handle;
};


struct READINST {
	OVERLAPPED ov;
	std::unique_ptr<char[]> buf ;
	HANDLE		hpipe;
	Process::Callback fn;
	READINST():buf(new char[BUFSIZE+1]), hpipe(INVALID_HANDLE_VALUE){
		memset(&ov,0,sizeof(ov));
		memset(buf.get(), 0, BUFSIZE+1);
	}
};


void wow64Check(bool enable) {
	typedef BOOL(APIENTRY *_Wow64EnableWow64FsRedirection)(BOOL);
	_Wow64EnableWow64FsRedirection  fun;
	HMODULE hmod = GetModuleHandleA("Kernel32.dll");
	fun = (_Wow64EnableWow64FsRedirection)GetProcAddress(hmod, "Wow64EnableWow64FsRedirection");
	if (fun) fun(enable);
};


VOID CALLBACK completionRoutine(
	DWORD dwErrorCode,
	DWORD dwNumberOfBytesTransfered,
	LPOVERLAPPED lpOverlapped ) {

	READINST* inst = (READINST*)lpOverlapped;
	if ( dwNumberOfBytesTransfered > 0 ) {
		inst->fn(inst->buf.get(), dwNumberOfBytesTransfered);
		memset(inst->buf.get(), 0, BUFSIZE);
		ReadFileEx(inst->hpipe, inst->buf.get(), BUFSIZE, lpOverlapped, completionRoutine);
	}
};


/// function CreatePipe can not work with readfileEx, so impl CreatePipeEx;
BOOL CreatePipeEx(PHANDLE  hReadPipe, PHANDLE  hWritePipe ) {
	
	UUID  uuid    = {0};
	UuidCreate(&uuid);

	char *pszuuid = NULL;
	UuidToStringA(&uuid, (RPC_CSTR*)&pszuuid);

	char  pipename[256] = { 0 };
	sprintf(pipename, "\\\\.\\pipe\\%s", pszuuid);
	RpcStringFreeA((RPC_CSTR*)&pszuuid);

	HANDLE read = CreateNamedPipeA(
		pipename, 
		PIPE_ACCESS_DUPLEX |FILE_FLAG_OVERLAPPED,
		PIPE_TYPE_BYTE |       
		PIPE_READMODE_BYTE | 
		PIPE_WAIT,
		1,
		4096,
		4096,
		0,                       
		NULL);

	if ( read == INVALID_HANDLE_VALUE ) {
		return FALSE;
	}


	SECURITY_ATTRIBUTES sa = {0};
	sa.nLength = sizeof(SECURITY_ATTRIBUTES);
	sa.bInheritHandle = TRUE;

	HANDLE write = CreateFileA(pipename, GENERIC_ALL, 0, &sa, OPEN_EXISTING, NULL, NULL);
	if ( write == INVALID_HANDLE_VALUE ) {
		CloseHandle(read);
		read = INVALID_HANDLE_VALUE;
		return FALSE;
	};

	*hReadPipe  = read;
	*hWritePipe = write;
	return TRUE;
}

Process::RunResult Process::syncRun(
	unsigned int timeout,Callback fstdout, Callback fstderr, int* exitCode ) {

	if ( exitCode ) {
		*exitCode = 0;
	}

	Handle stdout_rd, stdout_wr;
	Handle stderr_rd, stderr_wr;

	SECURITY_ATTRIBUTES sa;
	sa.nLength = sizeof(SECURITY_ATTRIBUTES);
	sa.bInheritHandle = TRUE;
	sa.lpSecurityDescriptor = nullptr;


	if ( fstdout  && !CreatePipeEx(&stdout_rd, &stdout_wr) ) {
		return RunResult::fail;
	}

	if ( fstderr && !CreatePipeEx(&stderr_rd, &stderr_wr) ) {
		return RunResult::fail;
	}


	PROCESS_INFORMATION process_info = {0};
	STARTUPINFOA        startup_info = {0};

	ZeroMemory(&process_info, sizeof(PROCESS_INFORMATION));

	ZeroMemory(&startup_info, sizeof(STARTUPINFO));
	startup_info.cb = sizeof(STARTUPINFO);
	startup_info.hStdOutput = stdout_wr;
	startup_info.hStdError  = stderr_wr;
	if (fstderr != nullptr || fstdout != nullptr)
		startup_info.dwFlags |= STARTF_USESTDHANDLES;

	wow64Check(false);
	BOOL bSuccess = CreateProcessA(nullptr,&m_command[0], nullptr, nullptr, TRUE, 0,
		nullptr, m_path.size()? &m_path[0]: nullptr, &startup_info, &process_info);
	wow64Check(true);

	if (!bSuccess)
		return fail;
	
	//为了自动关闭;	
	Handle hthread(process_info.hThread);
	Handle hprocess(process_info.hProcess); 

	if ( timeout == 0 ) {
		return RunResult::sucess;
	 }

	READINST errInst;
	if ( fstderr != nullptr ) {
		errInst.hpipe = stderr_rd;
		errInst.fn = fstderr;
		ReadFileEx(stderr_rd,
			errInst.buf.get(), BUFSIZE, (OVERLAPPED*)&errInst, completionRoutine);
	};

	READINST outInst;
	if ( fstdout != nullptr ) {
		outInst.hpipe = stdout_rd;
		outInst.fn = fstdout;
	    ReadFileEx(stdout_rd,
			outInst.buf.get(), BUFSIZE, (OVERLAPPED*)&outInst, completionRoutine);
	};

	time_t   start   = time(0);
	time_t   now     = start;
	unsigned int surplus = timeout; //超时单位 秒
	Process::RunResult  result = RunResult::sucess;
	bool     loop      = true;

	while ( loop ) {
		DWORD status = WaitForSingleObjectEx(process_info.hProcess, surplus*1000, TRUE);
		switch ( status ) {
		case WAIT_IO_COMPLETION:
			now = time(0);
			if ( now - start < timeout ) { //没有超时继续
				surplus = timeout - (now - start);
				break;
			}
			else {
				// time out next label
			}
		case WAIT_TIMEOUT:
			TerminateProcess(hprocess,0);
			result =  RunResult::timeout;
			loop = false;
			break;
		case WAIT_OBJECT_0:
			int retValue;
			GetExitCodeProcess(hprocess, (LPDWORD)&retValue);
			if ( exitCode ) {
				*exitCode = retValue;
			}
			result =  RunResult::sucess;
			loop = false;
		}
	}
	outInst.hpipe = INVALID_HANDLE_VALUE;
	errInst.hpipe = INVALID_HANDLE_VALUE;
	
	if (fstdout) CancelIoEx(stdout_rd,NULL); //cancel pending 
	if (fstderr) CancelIoEx(stderr_rd,NULL);
	
	SleepEx(1,TRUE); // fire that finished and not called.  after this funcion fire, will crash in completionRoutine ,

	return result;
};

#else



int  doRead(int fd, Process::Callback routine) {
	auto buffer = std::unique_ptr<char[]>(new char[BUFSIZE + 1]);
	memset(buffer.get(), 0, BUFSIZE);
	int n = read(fd, buffer.get(), BUFSIZE);
	if (n > 0 && routine) {
		routine(buffer.get(), static_cast<size_t>(n));
	}
	return n;
}


Process::RunResult Process::syncRun(
	unsigned int timeout,Callback fstdout, Callback fstderr, int* exitCode) {

	if ( exitCode ) {
		*exitCode = 0;
	}

	int  stdout_p[2] = { 0 }, stderr_p[2] = { 0 };

	if (  pipe(stdout_p) != 0 ) {
		return RunResult::fail;
	}
	if (  pipe(stderr_p) != 0 ) {
		 close(stdout_p[0]); close(stdout_p[1]); 
		return RunResult::fail;
	}

	int pid = fork();

	if ( pid < 0 ) {
		close(stdout_p[0]); close(stdout_p[1]); 
		close(stderr_p[0]); close(stderr_p[1]); 
		return RunResult::fail;
	}
	else if ( pid == 0 ) {

		dup2(stdout_p[1], 1);
		dup2(stderr_p[1], 2);

		close(stdout_p[0]); close(stdout_p[1]); 
		close(stderr_p[0]); close(stderr_p[1]); 
		setpgid(0, 0);
		//加载命令
		if ( !m_path.empty() ) {
			chdir( m_path.c_str() ); //change dir
		}
		execl("/bin/sh", "sh", "-c", m_command.c_str(), nullptr);
		_exit(EXIT_FAILURE);
	}
	

	close(stdout_p[1]);
	close(stderr_p[1]);

	if ( timeout == 0 ) {
		close(stdout_p[0]);
		close(stderr_p[0]);
	}

	int   maxfd = max(stdout_p[0], stderr_p[0]) + 1;


	time_t     start     = time(0);
	time_t     now       = start;
	timeval    tv        = { 0 };
	fd_set     fdsr      = { 0 };
	RunResult  reuslt    = RunResult::sucess;
	size_t     surplus   = timeout; 
	int       exit_status;

	while ( true ) {

		FD_ZERO(&fdsr);
		FD_SET(stdout_p[0], &fdsr);
		FD_SET(stderr_p[0], &fdsr);
		tv.tv_sec = surplus;


		int ret = select(maxfd, &fdsr, nullptr, nullptr, &tv);
		if ( ret > 0 ) { //可读
			if ( FD_ISSET(stdout_p[0], &fdsr)  &&
				 doRead(stdout_p[0], fstdout) == 0 ) {
					break;
			};
			if ( FD_ISSET(stderr_p[0], &fdsr) && 
				 doRead(stderr_p[0], fstderr) == 0 ) {
					break;
			};
		}
		else if( ret == 0 ) { //超时
			kill(pid, SIGKILL);
			reuslt = RunResult::timeout;
		} 
		
		now = time(0);
		if ( time(0) - start < timeout ) { //没有超时继续
			 surplus = timeout - (now - start);
			 continue;
		}
		else {
			kill(pid, SIGKILL);
			reuslt = RunResult::timeout;
		}
	}

	int p = waitpid(pid, &exit_status, 0);
	if ( p == pid && exitCode ) {
    getExitCode(exit_status, exitCode);
	}

	close( stdout_p[0] );
	close( stderr_p[0] );
	return reuslt;
}

void Process::getExitCode(int status, int* exitCode) {
  if (WIFEXITED(status)) {
    *exitCode = WEXITSTATUS(status);
     Log::Info("shell script exit,  exit code: %d", *exitCode);
  }
  else if (WIFSIGNALED(status)) {
      *exitCode = -1;
	  Log::Info("shell script termination, signal number = %d", WTERMSIG(status));
  }
  else if (WIFSTOPPED(status)) {
      *exitCode = -1;
	  Log::Info("shell script stopped, signal number = %d", WIFSTOPPED(status));
  }
}

#endif