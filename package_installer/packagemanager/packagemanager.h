// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#ifndef PACKAGE_INSTALLER_PACKAGEMANAGER_PACKAGEMANAGER_H_
#define PACKAGE_INSTALLER_PACKAGEMANAGER_PACKAGEMANAGER_H_

#include <vector>
#include <string>
#include "./packageinfo.h"
#include "./dbmanager.h"

namespace alyun_assist_installer {
class PackageManager {
 public:
  PackageManager();
  virtual ~PackageManager();
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
  bool InstallAction(const PackageInfo& package_info);
  void UninstallAction(const PackageInfo& package_info);
  std::vector<PackageInfo> GetPackageInfo(const std::string& package_name,
      const std::string& package_version = "",
      const std::string& arch = "");
  std::string GetRequestString(const std::string& package_name,
      const std::string& package_version,
      const std::string& arch);
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
  bool InstallLatestVersion(const std::vector<PackageInfo>& package_infos,
    const std::string& package_name);
  const PackageInfo* GetLatestVersion(const std::vector<PackageInfo>& package_infos,
    const std::string& package_name);
  DBManager* db_manager;
};

}  // namespace alyun_assist_installer

#endif  // PACKAGE_INSTALLER_PACKAGEMANAGER_PACKAGEMANAGER_H_
