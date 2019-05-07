/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-09
Type: .cpp
Description: Provide functions to get file path
**************************************************************************/

#ifdef _WIN32
#include <Windows.h>
#include <shlobj.h>
#include <io.h>
#include <direct.h>
#else
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#endif // _WIN32


#include "FileUtil.h"
#include "AssistPath.h"
#include <string.h>




AssistPath::AssistPath() :AssistPath("") {
}

/*
*Summary: this is a constructor func
*Parameters: (string) relative_path
(this parameter is relative path, you can set it like this  eg: "file01" "file02" "file03"……)
*Return: null.
*/
AssistPath::AssistPath(string relative_path) {
  _root_path = GetRootPath();
  if(!relative_path.empty())
    _root_path += FileUtils::separator() + relative_path;
}

/*
*Summary: this is a destructor func
*/
AssistPath::~AssistPath() {}

/*
*Summary: to get curre dir with this program
*Parameters: null
*Return: (string) root path
*/
string AssistPath::GetCurrDir() {
  return _root_path;
}

bool AssistPath::GetDefaultUserDataDirectory(std::string& path) {
  std::string organizationName = "Aliyun";
  std::string appName = "assist";
#ifdef _WIN32
  char buffer[MAX_PATH + 1];
  if (SHGetFolderPathA(0, CSIDL_LOCAL_APPDATA, 0 /* hToken */, SHGFP_TYPE_CURRENT, buffer) == S_OK) {
    path = buffer;
    path += '\\' + organizationName;
    if (!FileUtils::fileExists(path.c_str())) {
      FileUtils::mkdir(path.c_str());
    }
    path += '\\' + appName;
    if (!FileUtils::fileExists(path.c_str())) {
      FileUtils::mkdir(path.c_str());
    }
    return true;
  } else {
    return false;
  }
#else
  std::string xdgDataHome = "";
  char *home;
  home = getenv("XDG_DATA_HOME");
  if((NULL == home) || (0 == strlen(home))) {
      xdgDataHome = "/home/.local/share";
  } else {
      xdgDataHome = home;
  }

  xdgDataHome += "/data/" + organizationName +'/' + appName;
  if (!FileUtils::fileExists(xdgDataHome.c_str())) {
    FileUtils::mkpath(xdgDataHome.c_str());
  }
  path = xdgDataHome;
  return true;
#endif
}

bool AssistPath::GetTmpPath(std::string& path) {
#ifdef _WIN32
  char temp_path[MAX_PATH + 1];
  DWORD path_len = ::GetTempPathA(MAX_PATH, temp_path);
  if (path_len >= MAX_PATH || path_len <= 0)
    return false;
  path = temp_path;
  return true;
#else
  path = "/tmp";
  return true;
#endif
}

/*
*Summary: to get config dir path
(it has server.text , address.text …… in config dir)
*Parameters: null
*Return: (string) config path
*/
string AssistPath::GetConfigPath() {
  return GetCommonPath("config");
}

/*
*Summary: to get work dir path
*Parameters: (string) subpath
(if subpath is not empty, judge subpath file exist or not exist, not exist creat subpath dir)
*Return: (string) work path/ work subpath
*/
string AssistPath::GetWorkPath(string subpath) {
  string path = GetCommonPath("work");
  if (subpath == "")  return path;
  path += FileUtils::separator() + subpath;
  MakeSurePath(path);
  return path;
}

/*
*Summary: to get log dir path
*Parameters: (string) subpath
(if subpath is not empty, judge subpath file exist or not exist, not exist creat subpath dir)
*Return: (string) log path/ log subpath
*/
string AssistPath::GetLogPath(string subpath) {
  string path = GetCommonPath("log");
  if ( subpath == "" )  return path;
  path += FileUtils::separator() + subpath;
  MakeSurePath(path);
  return path;
}

/*
*Summary: to get setup dir path
*Parameters: (string) subpath
(if subpath is not empty, judge subpath file exist or not exist, not exist creat subpath dir)
*Return: (string) setup path/ setup subpath
*/
string AssistPath::GetSetupPath(string subpath) {
  string path = GetCommonPath("setup");
  if (subpath == "")  return path;
  path += FileUtils::separator() + subpath;
  MakeSurePath(path);
  return path;
}

/*
*Summary: to get backup dir path
*Parameters: (string) subpath
(if subpath is not empty, judge subpath file exist or not exist, not exist creat subpath dir)
*Return: (string) backup path/ backup subpath
*/
string AssistPath::GetBackupPath(string subpath) {
  string path = GetCommonPath("backup");
  if (subpath == "")  return path;
  path += FileUtils::separator() + subpath;
  MakeSurePath(path);
  return path;
};

/*
*Summary: to get root path
*Parameters: null
*Return: (string) root path
*/
string AssistPath::GetRootPath() {
  string strpath;
#if defined _WIN32
  char ctemp[1024] = { 0 };
  GetModuleFileNameA(NULL, ctemp, 1024);
  char *pPath = strrchr(ctemp, '\\');
  *pPath = '\0';
  strpath = string(ctemp);
#else
  char pbuf[1024] = { 0 };
  int count = 1024;
  int i;
  int  nrslt = 0;
  nrslt = readlink("/proc/self/exe", pbuf, count);
  if (nrslt < 0 || (nrslt >= count - 1))
    return "";
  pbuf[nrslt] = '\0';
  for ( i = nrslt; i >= 0; i-- ) {
    if ( pbuf[i] == '/' ) {
      pbuf[i] = '\0';
      break;
    }
  }
  strpath = string(pbuf);
#endif // define
  return  strpath;
}

/*
*Summary: to get subpath in root dir
*Parameters: (string) filedirname
(judge filedirname file exist or not exist, not exist creat filedirname dir)
*Return: (string) file dir path
*/
string AssistPath::GetCommonPath(string filedirname) {
  string path = _root_path + FileUtils::separator() + filedirname;
  MakeSurePath(path);
  return path;
}

/*
*Summary: to make sure path
*Parameters: (string) filename
(judge filename file exist or not exist, not exist creat filename dir or file)
*Return: (bool) status
*/
bool AssistPath::MakeSurePath(string filename) {
  bool status = false;
  if ( !IsFileExist(filename) ) {
    status = CreateFolder(filename);
  }
  return status;
}

/*
*Summary: to judge file exist or not
*Parameters: (string) filename
*Return: (bool) status
*/
bool AssistPath::IsFileExist(string filename) {

  if (access(filename.c_str(), 0) != 0) {
    return false;
  }
  return true;
}

/*
*Summary: to creat folder
*Parameters: (string) filename
*Return: (bool) status
*/
bool AssistPath::CreateFolder(string filename) {

  if (access(filename.c_str(), 0) == 0)  {
    return true;
  }

  int flag;

#ifdef WIN32
  flag = _mkdir(filename.c_str());
#else
  flag = mkdir(filename.c_str(), 0777);
#endif

  if (flag == 0)
    return true;
  else
    return false;

}





