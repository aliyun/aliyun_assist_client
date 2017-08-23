// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "gtest/gtest.h"

#include <string>
#include <vector>
#include <algorithm>

#include "../package_installer/packagemanager/dbmanager.h"

TEST(TestDBManager, ReplaceInto) {
  alyun_assist_installer::DBManager* db_manager = new alyun_assist_installer::DBManager();
  alyun_assist_installer::PackageInfo info;
  info.package_id = "1000000";
  info.display_name = "test";
  info.display_version = "1.0.0";
  info.new_version = "2.0.0";
  info.publisher = "test";
  info.install_date = "";
  info.url = "test";
  info.MD5 = "test";
  info.arch = "test";
  std::vector<alyun_assist_installer::PackageInfo> package_infos;
  package_infos.push_back(info);
  db_manager->ReplaceInto(package_infos);

  std::vector<alyun_assist_installer::PackageInfo> packages =
      db_manager->GetPackageInfosById("1000000");

  EXPECT_EQ(packages.size() > 0, true);
}

TEST(TestDBManager, GetPackageInfosById) {
  alyun_assist_installer::DBManager* db_manager =
      new alyun_assist_installer::DBManager();

  std::vector<alyun_assist_installer::PackageInfo> packages =
      db_manager->GetPackageInfosById("1000000");

  delete db_manager;
  EXPECT_EQ(packages.size() > 0, true);
  if (packages.size() > 0) {
    EXPECT_EQ(packages[0].package_id == "1000000", true);
  }
}

TEST(TestDBManager, GetPackageInfos1) {
  alyun_assist_installer::DBManager* db_manager =
      new alyun_assist_installer::DBManager();

  std::vector<alyun_assist_installer::PackageInfo> packages =
      db_manager->GetPackageInfos("test", true);

  delete db_manager;
  EXPECT_EQ(packages.size() > 0, true);
  if (packages.size() > 0) {
    EXPECT_EQ(packages[0].display_name == "test", true);
  }
}

TEST(TestDBManager, GetPackageInfos2) {
  alyun_assist_installer::DBManager* db_manager = 
      new alyun_assist_installer::DBManager();

  std::vector<alyun_assist_installer::PackageInfo> packages =
      db_manager->GetPackageInfos("test", "1.0.0", "test");

  delete db_manager;
  EXPECT_EQ(packages.size() > 0, true);
  if (packages.size() > 0) {
    EXPECT_EQ(packages[0].display_name == "test", true);
    EXPECT_EQ(packages[0].display_version == "1.0.0", true);
    EXPECT_EQ(packages[0].arch == "test", true);
  }
}

TEST(TestDBManager, Delete) {
  alyun_assist_installer::DBManager* db_manager =
      new alyun_assist_installer::DBManager();
  db_manager->Delete("1000000");
  std::vector<alyun_assist_installer::PackageInfo> packages =
      db_manager->GetPackageInfos("1000000");

  delete db_manager;
  EXPECT_EQ(packages.size() == 0, true);
}




