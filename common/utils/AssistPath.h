/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .h
Description: Provide functions to get file path
**************************************************************************/

#ifndef PROJECT_ASSISTPATH_H_
#define PROJECT_ASSISTPATH_H_

#include <string>
#include <list>

using  std::string;
using  std::wstring;
using  std::list;

class AssistPath {
 public:
  AssistPath();
  AssistPath(string relative_path);

  ~AssistPath();
  string GetCurrDir();
  bool   GetTmpPath(std::string& path);
  bool   GetDefaultUserDataDirectory(std::string& path);
  string GetConfigPath();
  string GetWorkPath(string subpath = "");
  string GetLogPath(string subpath = "");
  string GetSetupPath(string subpath = "");
  string GetBackupPath(string subpath = "");
  string GetPluginPath();
  string GetCrossVersionWorkPath();
  string GetScriptPath();
  bool   MakeSurePath(string path);
  int CreateDirRecursive(const std::string &directoryPath);
#if defined _WIN32
  bool SetCurrentEnvPath();
#endif

 private:
  string _root_path;
  string GetRootPath();
  string GetCommonPath(string filedirname);
  bool   CreateFolder(string filename);
  bool   IsFileExist(string filename);
};

#endif // PROJECT_ASSISTPATH_H_




