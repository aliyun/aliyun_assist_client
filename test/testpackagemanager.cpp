// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "gtest/gtest.h"

#include <string>
#include <vector>
#include <algorithm>
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/http_request.h"

#include "../package_installer/packagemanager/packagemanager.h"

TEST(TestPackageManager, DownloadFile) {
  std::string url("http://100.81.152.153:6666/package/python3/3.6.1/x86/python-3.6.1.zip");
  std::string path;
  AssistPath path_service("");
  path = path_service.GetCurrDir();
  path += FileUtils::separator();
  path += "python-3.6.1.zip";
  HttpRequest::download_file(url, path);
  bool ret = FileUtils::fileExists(path.c_str());
  EXPECT_EQ(true, ret);
}

TEST(TestPackageManager, CheckMd5) {
  std::string cur_path;
  AssistPath path_service("");
  cur_path = path_service.GetCurrDir();
  cur_path += FileUtils::separator();
  cur_path += "python-3.6.1.zip";
  alyun_assist_installer::PackageManager package_mgr;
  bool ret = package_mgr.CheckMd5(cur_path, "39192e116dce49bbd05efeced7924bae");
  EXPECT_EQ(true, ret);
}

TEST(TestPackageManager, Unzip) {
  std::string cur_path, cur_dir;
  AssistPath path_service("");
  cur_dir = path_service.GetCurrDir();
  cur_path += cur_dir;
  cur_path += FileUtils::separator();
  cur_path += "python-3.6.1.zip";
  alyun_assist_installer::PackageManager package_mgr;
  bool ret = package_mgr.UnZip(cur_path, cur_dir);
  EXPECT_EQ(true, ret);
}

TEST(TestPackageManager, parse_response_string) {
  std::string response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://100.81.152.153:6666/package/python3/3.6.1/x86/python-3.6.1.zip\",\
      \"md5\":\"39192e116dce49bbd05efeced7924bae\",\"version\":\"3.6.1\",\
      \"publisher\":\"Python Software Foundation\",\"arch\":\"x86\"}]";
  alyun_assist_installer::PackageManager package_mgr;
  vector<alyun_assist_installer::PackageInfo> package_infos =
      package_mgr.ParseResponseString(response);

  EXPECT_EQ(package_infos.size() > 0, true);
  if (package_infos.size() > 0) {
    EXPECT_EQ(package_infos[0].display_name == "python3", true);
    EXPECT_EQ(package_infos[0].display_version == "3.6.1", true);
    EXPECT_EQ(package_infos[0].display_version, "3.6.1");
    EXPECT_EQ(package_infos[0].url, "http://100.81.152.153:6666/package/python3/3.6.1/x86/python-3.6.1.zip");
    EXPECT_EQ(package_infos[0].MD5, "39192e116dce49bbd05efeced7924bae");
    EXPECT_EQ(package_infos[0].publisher, "Python Software Foundation");
    EXPECT_EQ(package_infos[0].arch, "x86");
  }
}

TEST(TestPackageManager, InstallAction) {
  std::string response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://100.81.152.153:6666/package/python3/3.6.1/x86/python-3.6.1.zip\",\
      \"md5\":\"39192e116dce49bbd05efeced7924bae\",\"version\":\"3.6.1\",\
      \"publisher\":\"Python Software Foundation\",\"arch\":\"x86\"}]";
  alyun_assist_installer::PackageManager package_mgr;
  vector<alyun_assist_installer::PackageInfo> package_infos =
      package_mgr.ParseResponseString(response);

  EXPECT_EQ(package_infos.size() > 0, true);
  if (package_infos.size() > 0)
  {
    package_mgr.InstallAction(package_infos[0]);
    alyun_assist_installer::DBManager* db_manager =
        new alyun_assist_installer::DBManager();
    std::vector<alyun_assist_installer::PackageInfo> packages =
        db_manager->GetPackageInfos("python3", "3.6.1");

    delete db_manager;
    EXPECT_EQ(packages.size() > 0, true);
    if (packages.size() > 0) {
      EXPECT_EQ(packages[0].display_name == "python3", true);
      EXPECT_EQ(packages[0].display_version == "3.6.1", true);
      EXPECT_EQ(packages[0].display_version, "3.6.1");
      EXPECT_EQ(packages[0].url, "http://100.81.152.153:6666/package/python3/3.6.1/x86/python-3.6.1.zip");
      EXPECT_EQ(packages[0].MD5, "39192e116dce49bbd05efeced7924bae");
      EXPECT_EQ(packages[0].publisher, "Python Software Foundation");
      EXPECT_EQ(packages[0].arch, "x86");
    }
  }
}

TEST(TestPackageManager, UninstallAction) {
  std::string response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://100.81.152.153:6666/package/python3/3.6.1/x86/python-3.6.1.zip\",\
      \"md5\":\"39192e116dce49bbd05efeced7924bae\",\"version\":\"3.6.1\",\
      \"publisher\":\"Python Software Foundation\",\"arch\":\"x86\"}]";
  alyun_assist_installer::PackageManager package_mgr;
  vector<alyun_assist_installer::PackageInfo> package_infos =
      package_mgr.ParseResponseString(response);

  EXPECT_EQ(package_infos.size() > 0, true);
  if (package_infos.size() > 0)
  {
    package_mgr.UninstallAction(package_infos[0]);
    alyun_assist_installer::DBManager* db_manager =
        new alyun_assist_installer::DBManager();
    std::vector<alyun_assist_installer::PackageInfo> packages =
        db_manager->GetPackageInfos("python3", "3.6.1");

    delete db_manager;
    EXPECT_EQ(packages.size() == 0, true);
  }
}
