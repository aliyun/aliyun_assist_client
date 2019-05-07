#pragma once
#include <string>
#include <functional>

using std::string;
using std::function;

class Process {
public:
	enum  RunResult {
		sucess,
		fail,
		timeout
	};

	typedef std::function<void(const char* bytes, size_t n)> Callback;

public:
	Process(string command, string path=string());
	RunResult syncRun(unsigned int timeout = 3600,Callback fstdout = nullptr, Callback fstderr = nullptr, int* exitCode = nullptr);
private:
#ifdef _WIN32
#else
  void getExitCode(int status, int* exitCode);
#endif

	string m_command;
  string m_path;
};

