// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#ifndef CLIENT_PACKAGE_INSTALLER_PACKAGEMANAGER_H_
#define CLIENT_PACKAGE_INSTALLER_PACKAGEMANAGER_H_

#include <vector>
#include <string>
#include "./packageinfo.h"
#include "./dbmanager.h"

namespace alyun_assist_installer {
class PackageManager {
 public:
  PackageManager();
  virtual ~PackageManager();
  static int CompareVersions(const std::string& a, const std::string& b);
  void List(const std::string& package_name);
  void Local(const std::string& package_name);
  void Latest(const std::string& package_name);
  void Install(const std::string& package_name,
      const std::string& package_version,
      const std::string& arch);
  void Uninstall(const std::string& package_name);
  void Update(const std::string& package_name);

#if defined(TEST_MODE)
 public:
#else
 private:
#endif
  void CheckInstall(const PackageInfo& package_info);
  void InstallAction(const PackageInfo& package_info);
  void UninstallAction(const PackageInfo& package_info);
  std::vector<PackageInfo> GetPackageInfo(const std::string& package_name,
      const std::string& package_version = "",
      const std::string& arch = "");
  std::string GetRequestString(const std::string& package_name,
      const std::string& package_version);
  std::vector<PackageInfo> ParseResponseString(std::string response);
  bool Download(const std::string& url, const std::string& path);
  bool CheckMd5(const std::string& path, const std::string& md5_string);
  bool UnZip(const std::string& file_name, const std::string& dir);
#ifdef _WIN32
  int ExecuteCmd(char* cmd, std::string& out);
#else
  int ExecuteCmd(char* cmd, char* buff, int size);
#endif
  int ComputeFileMD5(const std::string& file_path, std::string& md5_str);
  DBManager* db_manager;
};

}  // namespace alyun_assist_installer

#endif  // CLIENT_PACKAGE_INSTALLER_PACKAGEMANAGER_H_
