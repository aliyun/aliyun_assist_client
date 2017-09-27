// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./dbmanager.h"
#include <string>
#include <vector>
#include "./packageinfo.h"
#include "./utils/Log.h"
#include "./utils/AssistPath.h"
#include "./utils/FileUtil.h"

namespace alyun_assist_installer {
DBManager::DBManager() {
  Open();
  IsTableExist("package_infos");
}

DBManager::~DBManager() {
  sqlite3_close(db);
}

void DBManager::ReplaceInto(std::vector<PackageInfo> package_infos) {
  std::string sql;

  /* Create SQL statement */
  std::string replace_into = "REPLACE INTO package_infos " \
      "(package_id,display_name,display_version," \
      "publisher,url,md5,arch,install_date) ";
  size_t count = package_infos.size();
  for (int i = 0; i < count; ++i) {
    sql += replace_into;
    sql = sql + "VALUES (\"" + package_infos[i].package_id + "\",\"" +
        package_infos[i].display_name + "\",\"" +
        package_infos[i].display_version + "\",\"" +
        package_infos[i].publisher + "\",\"" +
        package_infos[i].url + "\",\"" +
        package_infos[i].MD5 + "\",\"" +
        package_infos[i].arch + "\"," +
      "datetime(\'now\') )";
  }

  Exec(sql.c_str(), NULL, NULL);
}

void DBManager::Delete(const std::string package_id) {
  std::string sql;

  /* Create merged SQL statement */
  sql = "DELETE from package_infos where package_id=\"" + package_id + "\"";
  Exec(sql.c_str(), NULL, NULL);
}

std::vector<PackageInfo> DBManager::GetPackageInfosById(const std::string package_id) {
  std::vector<PackageInfo> package_infos;
  std::string sql;

  /* Create SQL statement */
  sql = "SELECT * from package_infos where package_id=\"" + package_id + "\"";

  sqlite3_stmt * stmt = NULL;
  const char *zTail;
  if (sqlite3_prepare_v2(db, sql.c_str(), -1, &stmt, &zTail) == SQLITE_OK) {
    while (sqlite3_step(stmt) == SQLITE_ROW) {
      PackageInfo package;
      package.package_id = (char*)sqlite3_column_text(stmt, 0);
      package.display_name = (char*)sqlite3_column_text(stmt, 1);
      package.display_version = (char*)sqlite3_column_text(stmt, 2);
      package.publisher = (char*)sqlite3_column_text(stmt, 3);
      package.url = (char*)sqlite3_column_text(stmt, 4);
      package.MD5 = (char*)sqlite3_column_text(stmt, 5);
      package.arch = (char*)sqlite3_column_text(stmt, 6);
      package.install_date = (char*)sqlite3_column_text(stmt, 7);
      package_infos.push_back(package);
    }
  }

  sqlite3_finalize(stmt);
  return package_infos;
}

std::vector<PackageInfo> DBManager::GetPackageInfos(
    const std::string display_name, bool accurate) {
  std::vector<PackageInfo> package_infos;
  std::string sql;

  /* Create SQL statement */
  if (display_name.empty()) {
    sql = "SELECT * from package_infos";
  } else {
    if (accurate)
      sql = "SELECT * from package_infos where display_name=\"" +
          display_name + "\"";
    else
      sql = "SELECT * from package_infos where display_name like \"%" +
          display_name + "%\"";
  }

  sqlite3_stmt * stmt = NULL;
  const char *zTail;
  if (sqlite3_prepare_v2(db, sql.c_str(), -1, &stmt, &zTail) == SQLITE_OK) {
    while (sqlite3_step(stmt) == SQLITE_ROW) {
      PackageInfo package;
      package.package_id = (char*)sqlite3_column_text(stmt, 0);
      package.display_name = (char*)sqlite3_column_text(stmt, 1);
      package.display_version = (char*)sqlite3_column_text(stmt, 2);
      package.publisher = (char*)sqlite3_column_text(stmt, 3);
      package.url = (char*)sqlite3_column_text(stmt, 4);
      package.MD5 = (char*)sqlite3_column_text(stmt, 5);
      package.arch = (char*)sqlite3_column_text(stmt, 6);
      package.install_date = (char*)sqlite3_column_text(stmt, 7);
      package_infos.push_back(package);
    }
  }

  sqlite3_finalize(stmt);
  return package_infos;
}

std::vector<PackageInfo> DBManager::GetPackageInfos(
    const std::string display_name,
    const std::string package_version,
    const std::string arch) {
  std::vector<PackageInfo> package_infos;
  std::string sql;

  /* Create SQL statement */
  if (display_name.empty()) {
    sql = "SELECT * from package_infos";
  } else {
    sql = "SELECT * from package_infos where display_name like \"" +
        display_name + "\"";
    if (!package_version.empty()) {
      sql = sql + " and display_version=\"" + package_version + "\"";
    }
    if (!arch.empty()) {
      sql = sql + " and arch=\"" + arch + "\"";
    }
  }

  sqlite3_stmt * stmt = NULL;
  const char *zTail;
  if (sqlite3_prepare_v2(db, sql.c_str(), -1, &stmt, &zTail) == SQLITE_OK) {
    while (sqlite3_step(stmt) == SQLITE_ROW) {
      PackageInfo package;
      package.package_id = (char*)sqlite3_column_text(stmt, 0);
      package.display_name = (char*)sqlite3_column_text(stmt, 1);
      package.display_version = (char*)sqlite3_column_text(stmt, 2);
      package.publisher = (char*)sqlite3_column_text(stmt, 3);
      package.url = (char*)sqlite3_column_text(stmt, 4);
      package.MD5 = (char*)sqlite3_column_text(stmt, 5);
      package.arch = (char*)sqlite3_column_text(stmt, 6);
      package.install_date = (char*)sqlite3_column_text(stmt, 7);
      package_infos.push_back(package);
    }
  }

  sqlite3_finalize(stmt);
  return package_infos;
}

void DBManager::Open() {
  char *zErrMsg = 0;
  int  rc;
  AssistPath path_service("");
  std::string userdata_path = "";
  if (!path_service.GetDefaultUserDataDirectory(userdata_path)) {
    Log::Error("Get user data dir failed");
    return;
  }

  userdata_path += FileUtils::separator();
  userdata_path += "packageinfo.db";

  /* Open database */
  rc = sqlite3_open(userdata_path.c_str(), &db);
  if (rc) {
    Log::Error("Can't open database: %s\n", sqlite3_errmsg(db));
    exit(0);
  } else {
    // fprintf(stdout, "Opened database successfully\n");
  }
}

void DBManager::Exec(const char *sql,
    int(*callback)(void*, int, char**, char**),
    void * data) {
  char *zErrMsg = 0;
  int  rc;

  /* Execute SQL statement */
  rc = sqlite3_exec(db, sql, NULL, data, &zErrMsg);
  if (rc != SQLITE_OK) {
    Log::Error("SQL error, sql: %s, error: %s\n", sql, zErrMsg);
    sqlite3_free(zErrMsg);
  } else {
    // fprintf(stdout, "Operation done successfully\n");
  }
}

static int callback(void *data, int argc, char **argv, char **azColName) {
  if (argc == 1) {
    if (atoi(argv[0]) == 1) {
      return 0;
    } else {
      ((DBManager*)data)->CreateTable();
    }
  }
  return 0;
}

void DBManager::IsTableExist(const char * table_name) {
  std::string sql;

  /* Create SQL statement */
  sql = "SELECT COUNT(*) FROM sqlite_master where type='table' and name=" +
    std::string("\'") + table_name + std::string("\'");

  /* Execute SQL statement */
  sqlite3_exec(db, sql.c_str(), callback, (void*)this, NULL);
}

void DBManager::CreateTable() {
  char *zErrMsg = 0;
  int  rc;
  char *sql;

  /* Create SQL statement */
  sql = "CREATE TABLE package_infos("  \
      "package_id INT PRIMARY KEY     NOT NULL," \
      "display_name           TEXT    NOT NULL," \
      "display_version        TEXT    NOT NULL," \
      "publisher              TEXT    NOT NULL," \
      "url                    TEXT    NOT NULL," \
      "md5                    TEXT    NOT NULL," \
      "arch                    TEXT    NOT NULL," \
      "install_date           datetime     NOT NULL );";

  /* Execute SQL statement */
  rc = sqlite3_exec(db, sql, 0, 0, &zErrMsg);
  if (rc != SQLITE_OK) {
    // fprintf(stderr, "SQL error: %s\n", zErrMsg);
    Log::Error("create table failed sql: %s, error: %s", sql, zErrMsg);
    sqlite3_free(zErrMsg);
  } else {
    // fprintf(stdout, "Table created successfully\n");
  }
}
}  // namespace alyun_assist_installer
