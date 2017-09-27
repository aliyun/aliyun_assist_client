// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./packagemanager.h"
#include <string>
#include <vector>
#include <algorithm>
#ifdef _WIN32
#include <windows.h>
#else
#include <unistd.h>
#include <sys/wait.h>
#endif
#include "./packageinfo.h"
#include "utils/AssistPath.h"
#include "utils/OsVersion.h"
#include "utils/FileVersion.h"
#include "utils/CheckNet.h"
#include "utils/http_request.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"
#include "utils/VersionComparator.h"
#include "jsoncpp/json.h"
#include "zip/zip.h"
#include "md5/md5.h"

namespace alyun_assist_installer {
PackageManager::PackageManager() {
  db_manager = new DBManager();
}

PackageManager::~PackageManager() {
  delete db_manager;
}

void PackageManager::List(const std::string& package_name) {
  Log::Info("Enter list, package_name: %s", package_name.c_str());
  vector<PackageInfo> package_infos = GetPackageInfo(package_name);

  if (package_infos.empty()) {
    if (package_name.empty()) {
      Log::Error("There is no package in the software repository.");
      printf("There is no package in the software repository.\n");
    } else {
      Log::Info("There is no package named %s in the software repository",
          package_name.c_str());
      printf("There is no package named %s in the software repository\n",
          package_name.c_str());
    }
  } else {
    printf("package_id\tname\tversion\tarch\tpublisher\n");
    for (size_t i = 0; i < package_infos.size(); ++i) {
      printf("%s\t%s\t%s\t%s\t%s\n", package_infos[i].package_id.c_str(),
        package_infos[i].display_name.c_str(),
        package_infos[i].display_version.c_str(),
        package_infos[i].arch.c_str(),
        package_infos[i].publisher.c_str());
    }
  }
}

void PackageManager::Local(const std::string& package_name) {
  Log::Info("Enter Local, package_name: %s", package_name.c_str());
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, false);

  if (package_infos.empty()) {
    if (package_name.empty()) {
      Log::Info("There is no package in the local");
      printf("There is no package in the local\n");
    } else {
      Log::Info("There is no package named %s in the local",
          package_name.c_str());
      printf("There is no package named %s in the local\n",
          package_name.c_str());
    }
  } else {
    printf("package_id\tname\tversion\tarch\tpublisher\tinstall date\n");
    for (size_t i = 0; i < package_infos.size(); ++i) {
      printf("%s\t%s\t%s\t%s\t%s\t%s\n", package_infos[i].package_id.c_str(),
        package_infos[i].display_name.c_str(),
        package_infos[i].display_version.c_str(),
        package_infos[i].arch.c_str(),
        package_infos[i].publisher.c_str(),
        package_infos[i].install_date.c_str());
    }
  }
}

void PackageManager::Latest(const std::string& package_name) {
  Log::Info("Enter Latest, package_name: %s", package_name.c_str());
  // query the package in the local
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, false);

  vector<PackageInfo> new_packages;
  for (size_t i = 0; i < package_infos.size(); ++i) {
    package_infos[i].new_version = package_infos[i].display_version;
    vector<PackageInfo> packages =
        GetPackageInfo(package_infos[i].display_name);
    for (size_t j = 0; j < packages.size(); j++) {
      if ((package_infos[i].display_name == packages[j].display_name) &&
          (package_infos[i].arch == packages[j].arch)) {
        // compare the version of the local package and remote package
        if (VersionComparator::CompareVersions(packages[j].display_version,
            package_infos[i].new_version) > 0) {
          package_infos[i].new_version = packages[j].display_version;
        }
      }
    }
  }

  for (size_t i = 0; i < package_infos.size(); ++i) {
    if (package_infos[i].new_version != package_infos[i].display_version) {
      new_packages.push_back(package_infos[i]);
    }
  }

  if (!new_packages.empty()) {
    printf("name\tversion\tnewversion\tarch\tpublisher\n");
    for (size_t i = 0; i < new_packages.size(); ++i) {
      printf("%s\t%s\t%s\t%s\t%s\n", new_packages[i].display_name.c_str(),
          new_packages[i].display_version.c_str(),
          new_packages[i].new_version.c_str(),
          new_packages[i].arch.c_str(),
          new_packages[i].publisher.c_str());
    }
  }
}

void PackageManager::Install(const std::string& package_name,
    const std::string& package_version,
    const std::string& arch) {
  Log::Info("Enter Install, package_name: %s, package_version: %s,arch: %s",
      package_name.c_str(), package_version.c_str(), arch.c_str());

  // If the package_version is empty, fuzzy query the package_name
  if (package_version.empty()) {
    vector<PackageInfo> package_infos = GetPackageInfo(package_name);
    if (package_infos.empty()) {
      Log::Info("There is no package named %s in the software repository",
        package_name.c_str());
      printf("There is no package named %s in the software repository\n",
        package_name.c_str());
      return;
    } else {
      printf("package_id\tname\tversion\tarch\tpublisher\n");
      for (size_t i = 0; i < package_infos.size(); ++i) {
        printf("%s\t%s\t%s\t%s\t%s\n", package_infos[i].package_id.c_str(),
            package_infos[i].display_name.c_str(),
            package_infos[i].display_version.c_str(),
            package_infos[i].arch.c_str(),
            package_infos[i].publisher.c_str());
      }
    }

    // If there are many packages whose name include package_name,
    // ask user to input the package_id
    while (true) {
      printf("Please input the package_id you want to install.\n");
      char package_id[100];
      scanf("%s", package_id);
      int dest_package_index = -1;
      printf("The package_id you input is %s.\n", package_id);
      for (size_t i = 0; i < package_infos.size(); ++i) {
        if (package_infos[i].package_id == package_id) {
          dest_package_index = i;
          break;
        }
      }

      if (dest_package_index > -1 &&
          dest_package_index < package_infos.size()) {
        printf("The package you want to install is %s.\n",
            package_infos[dest_package_index].display_name.c_str());
        CheckInstall(package_infos[dest_package_index]);
        break;
      } else {
        printf("The package_id you input is not exist.\n");
      }
    }
  } else {
    // If the package_version is not empty, accurate query the package_name
    vector<PackageInfo> package_infos =
        GetPackageInfo(package_name, package_version, arch);
    if (package_infos.empty()) {
      printf("There is no such package.\n");
    } else {
      CheckInstall(package_infos[0]);
    }
  }
}

void PackageManager::Uninstall(const std::string& package_name) {
  Log::Info("Enter Uninstall, package_name: %s", package_name.c_str());
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, true);
  if (!package_infos.empty()) {
    UninstallAction(package_infos[0]);
  } else {
    Log::Info("Uninstall failed, no package named %s", package_name.c_str());
    printf("There is no such package.\n");
    return;
  }
}

void PackageManager::Update(const std::string& package_name) {
  Log::Info("Enter Update, package_name: %s", package_name.c_str());
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, true);
  if (package_infos.empty()) {
    Log::Info("There is no package named %s in the local",
        package_name.c_str());
    printf("There is no such package on this machine.\n");
    return;
  }

  std::string new_version = package_infos[0].display_version;
  vector<PackageInfo> packages = GetPackageInfo(package_infos[0].display_name);
  int index = -1;
  for (size_t i = 0; i < packages.size(); i++) {
    if ((package_infos[0].display_name == packages[i].display_name) &&
        (package_infos[0].arch == packages[i].arch)) {
      // compare the version of the local package and remote package
      if (VersionComparator::CompareVersions(
          packages[i].display_version, new_version) > 0) {
        new_version = packages[i].display_version;
        index = i;
      }
    }
  }

  if (index == -1) {
    Log::Info("The package is latest. There is no need to update, %s",
        package_name.c_str());
    printf("The package is latest. There is no need to update.\n");
    return;
  }

  if (InstallAction(packages[index]))
    db_manager->Delete(package_infos[0].package_id);
}

void PackageManager::CheckInstall(const PackageInfo& package_info) {
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfosById(package_info.package_id);

  if (!package_infos.empty()) {
    printf("name\tversion\tpublisher\tinstall data\n");
    for (size_t i = 0; i < package_infos.size(); ++i) {
      printf("%s\t%s\t%s\t%s\n", package_infos[i].display_name.c_str(),
          package_infos[i].display_version.c_str(),
          package_infos[i].publisher.c_str(),
          package_infos[i].install_date.c_str());
      Log::Info("This package is already exist, %s.",
          package_infos[i].display_name.c_str());
      printf("This package is already exist.\n");
    }

    return;
  }

  InstallAction(package_info);
}

bool PackageManager::InstallAction(const PackageInfo& package_info) {
  Log::Info("Enter InstallAction");
  AssistPath path("");
  std::string userdata_path = "";
  path.GetDefaultUserDataDirectory(userdata_path);
  Log::Info("userdata_path, %s", userdata_path.c_str());
  std::string file_path = userdata_path;
  std::string file_name = package_info.url.substr(
      package_info.url.find_last_of('/') + 1);
  file_path += FileUtils::separator();
  file_path.append(file_name);
  Log::Info("Call Download, %s", package_info.url.c_str());
#if !defined(TEST_MODE)
  printf("Downloading...\n");
  bool download_ret = Download(package_info.url, file_path);
  if (!download_ret) {
    printf("Download this package failed, please try again later.\n");
    return false;
  }

  printf("Check MD5\n");
  if (!CheckMd5(file_path, package_info.MD5)) {
    printf("Check file md5 failed.\n");
    return false;
  }

  printf("Unzip\n");
  bool unzip_ret = UnZip(file_path, userdata_path);
  if (!unzip_ret) {
    printf("Unzip this package failed, please try again later.\n");
    return false;
  }
#endif

  printf("Installing...\n");
  std::string install_dir = userdata_path;
  std::string cmd = "";
#ifdef _WIN32
  install_dir.append("\\");
  install_dir.append(file_name.substr(0, file_name.find_last_of('.')));
  std::string install_file = install_dir;
  install_file.append("\\");
  install_file.append("install.bat");
  cmd = install_file + " " + install_dir;
#else
  install_dir.append("/");
  install_dir.append(file_name.substr(0, file_name.find_last_of('.')));
  std::string install_file = install_dir;
  install_file.append("/");
  install_file.append("install.sh");
  cmd = "chmod 744 " + install_file;
  system(cmd.c_str());
  cmd = install_file;
#endif

#ifdef _WIN32
  std::string out;
  char srcipt_path[1024] = { 0 };
  strcpy(srcipt_path, cmd.c_str());
  int code = ExecuteCmd(srcipt_path, out);
  if (code == 0 && (out.find("Installation success") != string::npos)) {
    vector<PackageInfo> package_infos;
    package_infos.push_back(package_info);
    db_manager->ReplaceInto(package_infos);
    remove(file_path.c_str());
    printf("%s", out.c_str());
    return true;
  } else {
    Log::Info("Installation failed, %s.", out);
    printf("Installation failed.\n%s\n", out);
    return false;
  }
#else
  char srcipt_path[1024] = { 0 };
  char buf[10240] = { 0 };
  strcpy(srcipt_path, cmd.c_str());
  int code = ExecuteCmd(srcipt_path, buf, 10240);
  std::string out = buf;
  if (code == 0 && (out.find("Installation success") != string::npos)) {
    vector<PackageInfo> package_infos;
    package_infos.push_back(package_info);
    db_manager->ReplaceInto(package_infos);
    remove(file_path.c_str());
    printf("%s", buf);
    return true;
  } else {
    Log::Info("Installation failed, %s.", buf);
    printf("Installation failed.\n%s\n", buf);
    return false;
  }
#endif
}

void PackageManager::UninstallAction(const PackageInfo& package_info) {
  AssistPath path("");
  std::string userdata_path;
  path.GetDefaultUserDataDirectory(userdata_path);
  std::string file_name = package_info.url.substr(
      package_info.url.find_last_of('/') + 1);
  std::string uninstall_dir = userdata_path + FileUtils::separator();
  uninstall_dir.append(file_name.substr(0, file_name.find_last_of('.')));
  std::string uninstall_file = uninstall_dir;
  std::string cmd = uninstall_dir + FileUtils::separator();
#ifdef _WIN32
  cmd.append("uninstall.bat");
#else
  cmd.append("uninstall.sh");
  std::string chmod_cmd = "chmod 744 " + cmd;
  system(chmod_cmd.c_str());
#endif

  printf("Uninstalling...\n");
#ifdef _WIN32
  std::string out;
  char srcipt_path[1024] = { 0 };
  strcpy(srcipt_path, cmd.c_str());
  int code = ExecuteCmd(srcipt_path, out);
  if (code == 0 && (out.find("Uninstallation success") != string::npos)) {
    db_manager->Delete(package_info.package_id);
    printf("%s", out.c_str());
  } else {
    Log::Info("Uninstallation failed, %s.", out);
    printf("Uninstallation failed.\n%s\n", out);
  }
#else
  char srcipt_path[1024] = { 0 };
  char buf[10240] = { 0 };
  strcpy(srcipt_path, cmd.c_str());
  int code = ExecuteCmd(srcipt_path, buf, 10240);
  std::string out = buf;
  if (code == 0 && (out.find("Uninstallation success") != string::npos)) {
    db_manager->Delete(package_info.package_id);
    printf("%s", buf);
  } else {
    Log::Info("Uninstallation failed, %s.", buf);
    printf("Uninstallation failed.\n%s\n", buf);
  }
#endif
}

vector<PackageInfo> PackageManager::GetPackageInfo(
    const std::string& package_name,
    const std::string& package_version,
    const std::string& arch) {
  std::string json = GetRequestString(package_name, package_version, arch);
  std::string response;
  vector<PackageInfo> package_infos;

  if (HostChooser::m_HostSelect.empty()) {
    return package_infos;
  }

  std::string url = "http://" + HostChooser::m_HostSelect +
    "/luban/api/v1/repo/query_software?";
  /*std::string url = "http://100.81.152.153:6666";
  url += "/luban/api/v1/repo/query_software?";*/

  bool ret = HttpRequest::http_request_post(url, json, response);
  /*ret = true;
  response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://30.27.84.30:5656/python-3.6.1.zip\",\
      \"md5\":\"39192e116dce49bbd05efeced7924bae\",\"version\":\"3.6.1\",\
      \"publisher\":\"Python Software Foundation\",\"arch\":\"x86\"}]";*/

  Log::Info("response:%s", response.c_str());
  if (ret) {
    package_infos = ParseResponseString(response);
  } else {
    Log::Error("http request failed, url: %s, response:%s",
        url.c_str(), response.c_str());
  }

  return package_infos;
}

std::string PackageManager::GetRequestString(
    const std::string& package_name,
    const std::string& package_version,
    const std::string& arch) {
  Json::Value jsonRoot;
  if (!package_name.empty())
    jsonRoot["package_name"] = package_name;

  if (!package_version.empty())
    jsonRoot["package_version"] = package_version;

  if (!arch.empty())
    jsonRoot["arch"] = arch;

#ifdef _WIN32
  jsonRoot["os"] = "windows";
#else
  jsonRoot["os"] = "linux";
#endif

  return jsonRoot.toStyledString();
}

vector<PackageInfo> PackageManager::ParseResponseString(
    std::string response) {
  Json::Value jsonRoot;
  Json::Reader reader;

  vector<PackageInfo> package_infos;
  try {
    if (!reader.parse(response, jsonRoot)) {
      Log::Error("invalid json format");
      return package_infos;
    }

    for (size_t i = 0; i < jsonRoot.size(); ++i) {
      PackageInfo package_info;
      if (jsonRoot[i]["packageId"].isString())
        package_info.package_id = jsonRoot[i]["packageId"].asString();
      if (jsonRoot[i]["url"].isString())
        package_info.url = jsonRoot[i]["url"].asString();
      if (jsonRoot[i]["md5"].isString())
        package_info.MD5 = jsonRoot[i]["md5"].asString();
      if (jsonRoot[i]["name"].isString())
        package_info.display_name = jsonRoot[i]["name"].asString();
      if (jsonRoot[i]["version"].isString())
        package_info.display_version = jsonRoot[i]["version"].asString();
      if (jsonRoot[i]["publisher"].isString())
        package_info.publisher = jsonRoot[i]["publisher"].asString();
      if (jsonRoot[i]["arch"].isString())
        package_info.arch = jsonRoot[i]["arch"].asString();
      std::transform(package_info.MD5.begin(), package_info.MD5.end(),
        package_info.MD5.begin(), ::tolower);
      package_infos.push_back(package_info);
    }
  }
  catch (...) {
    Log::Error("ParseResponseString failed, response:%s",
        response.c_str());
  }

  return package_infos;
}

bool PackageManager::Download(const std::string& url,
    const std::string& path) {
  Log::Info("Enter Download, %s", url.c_str());
  bool ret = HttpRequest::download_file(url, path);
  if (ret) {
    return true;
  } else {
    Log::Error("Download failed, url: %s", url.c_str());
    return false;
  }
}

bool PackageManager::CheckMd5(const std::string& path,
    const std::string& md5_string) {
  std::string md5_str = md5_string;
  std::transform(md5_str.begin(), md5_str.end(),
      md5_str.begin(), ::tolower);
  std::string content;
  FileUtils::ReadFileToString(path, content);
  md5 md5_service(content);
  std::string file_md5;
  int code = ComputeFileMD5(path, file_md5);
  if (code == -1) {
    Log::Error("ComputeFileMD5 failed");
    return false;
  }

  std::transform(file_md5.begin(), file_md5.end(),
    file_md5.begin(), ::tolower);
  if (md5_str.compare(file_md5) == 0) {
    return true;
  } else {
    Log::Error("CheckMd5 failed, path: %s, file_md5: %s, md5_str: %s",
      path.c_str(), file_md5.c_str(), md5_str.c_str());
    return false;
  }
}

bool PackageManager::UnZip(const std::string& file_name,
    const std::string& dir) {
  int ret = zip_extract(file_name.c_str(), dir.c_str(), nullptr, nullptr);
  if (ret == 0) {
    return true;
  } else {
    Log::Error("UnZip failed, file name: %s", file_name.c_str());
    return false;
  }
}

#ifdef _WIN32
int PackageManager::ExecuteCmd(char* cmd, std::string& out) {
  DWORD exitCode = -1;
  SECURITY_ATTRIBUTES sattr = { 0 };

  sattr.nLength = sizeof(sattr);
  sattr.bInheritHandle = TRUE;

  HANDLE hChildOutR;
  HANDLE hChildOutW;
  if (!CreatePipe(&hChildOutR, &hChildOutW, &sattr, 0)) {
    exitCode = GetLastError();
    Log::Error("CreatePipe failed, url: %d", exitCode);
    return exitCode;
  }

  SetHandleInformation(hChildOutR, HANDLE_FLAG_INHERIT, 0);

  STARTUPINFOA si = { 0 };
  PROCESS_INFORMATION pi = { 0 };

  si.cb = sizeof(si);
  si.hStdOutput = hChildOutW;
  si.hStdError = hChildOutW;
  si.dwFlags |= STARTF_USESTDHANDLES;

  BOOL ret = FALSE;
  ret = CreateProcessA(NULL, cmd, 0, 0, TRUE, 0, 0, 0, &si, &pi);

  if (!ret) {
    exitCode = GetLastError();
    Log::Error("CreateProcessA failed, url: %d", exitCode);
    return exitCode;
  }

  DWORD dw = WaitForSingleObject(pi.hProcess, 60 * 60 * 1000);
  DWORD len = 0;
  CHAR  output[0x1000] = { 0 };
  switch (dw) {
  case WAIT_OBJECT_0:
    GetExitCodeProcess(pi.hProcess, &exitCode);
    PeekNamedPipe(hChildOutR, output, sizeof(output), 0, &len, 0);
    out = output;
    break;

  case WAIT_TIMEOUT:
    Log::Error("wait timeout: %d", GetLastError());
    exitCode = GetLastError();
    TerminateProcess(pi.hProcess, 1);

  case WAIT_FAILED:
    Log::Error("wait failed: %d", GetLastError());
    exitCode = GetLastError();
  }

  CloseHandle(hChildOutR);
  CloseHandle(hChildOutW);
  CloseHandle(pi.hProcess);
  CloseHandle(pi.hThread);

  return exitCode;
}
#else
int PackageManager::ExecuteCmd(char* cmd, char* buff, int size) {
  char temp[256];
  FILE* fp = NULL;
  int offset = 0;
  int len;

  fp = popen(cmd, "r");
  if (fp == NULL) {
    return -1;
  }

  while (fgets(temp, sizeof(temp), fp) != NULL) {
    len = strlen(temp);
    if (offset + len < size) {
      strcpy(buff+offset, temp);
      offset += len;
    } else {
      buff[offset] = 0;
      break;
    }
  }

  if (fp != NULL) {
    pclose(fp);
  }

  return 0;
}
#endif

int PackageManager::ComputeFileMD5(const std::string& file_path,
    std::string& md5_str) {
  md5 md5_service;

  FILE* file = fopen(file_path.c_str(), "rb");
  if (!file) {
    Log::Error("fopen failed, file_path: %s", file_path.c_str());
    return -1;
  }

  const size_t kBufferSize = 1 << 14;
  char* buf = new char[kBufferSize];
  size_t len;
  size_t size = 0;

  fseek(file, 0, SEEK_END);
  long totle_len = ftell(file);
  if (totle_len < 0) {
    fclose(file);
    Log::Error("ftell failed, file_path: %s", file_path.c_str());
    return -1;
  }

  fseek(file, 0, SEEK_SET);
  while ((len = fread(buf, sizeof(char), kBufferSize, file)) > 0) {
    md5_service.update(buf, len);
    size += len;
  }

  if (totle_len != size) {
    fclose(file);
    Log::Error("fread failed, totle_len: %d, read_size: %d",
        totle_len, size);
    return -1;
  }

  delete[] buf;
  fclose(file);

  md5_service.finalize();
  md5_str = md5_service.hexdigest();

  return 0;
}
}  // namespace alyun_assist_installer
