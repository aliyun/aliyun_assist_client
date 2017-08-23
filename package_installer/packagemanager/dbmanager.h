// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#ifndef CLIENT_PACKAGE_INSTALLER_DBMANAGER_H_
#define CLIENT_PACKAGE_INSTALLER_DBMANAGER_H_

#include <vector>
#include <string>
#include "./packageinfo.h"
#include "sqlite3/sqlite3.h"

namespace alyun_assist_installer {
class DBManager {
 public:
  DBManager();
  virtual ~DBManager();
  void ReplaceInto(std::vector<PackageInfo> package_infos);
  void Delete(const std::string package_id);
  std::vector<PackageInfo> GetPackageInfosById(const std::string package_id);
  std::vector<PackageInfo> GetPackageInfos(const std::string display_name,
      bool accurate = true);
  std::vector<PackageInfo> GetPackageInfos(const std::string display_name,
      const std::string package_version, const std::string arch = "");
  void CreateTable();
 private:
  void Open();
  void Exec(const char *sql, int(*callback)(void*, int, char**, char**),
    void * data);
  void IsTableExist(const char * table_name);
  sqlite3 *db;
};

}  // namespace alyun_assist_installer

#endif  // CLIENT_PACKAGE_INSTALLER_DBMANAGER_H_
