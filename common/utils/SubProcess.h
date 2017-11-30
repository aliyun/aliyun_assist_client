/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .h
Description: Provide functions to make process
**************************************************************************/

#ifndef PROJECT_SUBPROCESS_H_
#define PROJECT_SUBPROCESS_H_

#if defined(_WIN32)
#include "windows.h"
#endif
#include <string.h>
#include <stdio.h>
#include <iostream>
#include <stdlib.h>

using  std::string;
using  std::wstring;

class SubProcess {
 public:
  SubProcess(string cwd, int _time_out);
  SubProcess(string cmd, string cwd = "");
  ~SubProcess();
  bool Execute();
  void set_cmd(string cmd) {
    _cmd = cmd;
  }
#if defined(_WIN32)
  HANDLE get_id() {
    return _hProcess;
  }
#endif

  bool Execute(string &out, long &exitCode);
  bool RunModule(string moduleName);
  bool IsExecutorExist(string guid);

 private:
  string  _cmd;
  string  _cwd;
#if defined(_WIN32)
  HANDLE _hProcess;
#endif
  int  _time_out;
  bool ExecuteCmd(char * cmd, const char * cwd, bool isWait, string & out, long & exitCode);
  void EnableWow64(bool enable);
  bool ExecuteCMD_LINUX(char* cmd, const char* cwd, bool isWait, string& out, long &exitCode);
};

#endif //PROJECT_SUBPROCESS_H_

