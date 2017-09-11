// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "gtest/gtest.h"

#include <string>
#include <vector>
#include <algorithm>
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/http_request.h"

#include "../package_installer/packagemanager/packagemanager.h"

//TEST(TestPackageManager, DownloadFile) {
//  std::string url("http://www.test.com:6666/package/python3/3.6.1/x86/python-3.6.1.zip");
//  std::string path;
//  AssistPath path_service("");
//  path = path_service.GetCurrDir();
//  path += FileUtils::separator();
//  path += "python-3.6.1.zip";
//  HttpRequest::download_file(url, path);
//  bool ret = FileUtils::fileExists(path.c_str());
//  EXPECT_EQ(true, ret);
//}

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
#ifdef _WIN32
  cur_path += "python-3.6.1.zip";
#else
  cur_path += "node-v6.11.2-linux-x86.zip";
#endif
  alyun_assist_installer::PackageManager package_mgr;
  bool ret = package_mgr.UnZip(cur_path, cur_dir);
  EXPECT_EQ(true, ret);
}

TEST(TestPackageManager, parse_response_string) {
  std::string response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://www.test.com:6666/package/python3/3.6.1/x86/python-3.6.1.zip\",\
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
    EXPECT_EQ(package_infos[0].url, "http://www.test.com:6666/package/python3/3.6.1/x86/python-3.6.1.zip");
    EXPECT_EQ(package_infos[0].MD5, "39192e116dce49bbd05efeced7924bae");
    EXPECT_EQ(package_infos[0].publisher, "Python Software Foundation");
    EXPECT_EQ(package_infos[0].arch, "x86");
  }
}

TEST(TestPackageManager, InstallAction) {
  std::string response = "";
#ifdef _WIN32
  response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://www.test.com:6666/package/python3/3.6.1/x86/python-3.6.1.zip\",\
      \"md5\":\"39192e116dce49bbd05efeced7924bae\",\"version\":\"3.6.1\",\
      \"publisher\":\"Python Software Foundation\",\"arch\":\"x86\"}]";
#else
  response = "[{\"packageId\":\"1\",\"name\":\"nodejs\",\
      \"url\":\"http://www.test.com:6666/package/nodejs/6.11.2/x86/node-v6.11.2-linux-x86.zip\",\
      \"md5\":\"test\",\"version\":\"6.11.2\",\
      \"publisher\":\"test\",\"arch\":\"x86\"}]";
#endif
  alyun_assist_installer::PackageManager package_mgr;
  vector<alyun_assist_installer::PackageInfo> package_infos =
      package_mgr.ParseResponseString(response);

  EXPECT_EQ(package_infos.size() > 0, true);
  if (package_infos.size() > 0)
  {
    package_mgr.InstallAction(package_infos[0]);
    alyun_assist_installer::DBManager* db_manager =
        new alyun_assist_installer::DBManager();

#ifdef _WIN32
    std::vector<alyun_assist_installer::PackageInfo> packages =
        db_manager->GetPackageInfos("python3", "3.6.1");

    delete db_manager;
    EXPECT_EQ(packages.size() > 0, true);
    if (packages.size() > 0) {
      EXPECT_EQ(packages[0].display_name, "python3");
      EXPECT_EQ(packages[0].display_version, "3.6.1");
      EXPECT_EQ(packages[0].url, "http://www.test.com:6666/package/python3/3.6.1/x86/python-3.6.1.zip");
      EXPECT_EQ(packages[0].MD5, "39192e116dce49bbd05efeced7924bae");
      EXPECT_EQ(packages[0].publisher, "Python Software Foundation");
      EXPECT_EQ(packages[0].arch, "x86");
    }
#else
    std::vector<alyun_assist_installer::PackageInfo> packages =
      db_manager->GetPackageInfos("nodejs", "6.11.2");

    delete db_manager;
    EXPECT_EQ(packages.size() > 0, true);
    if (packages.size() > 0) {
      EXPECT_EQ(packages[0].display_name, "nodejs");
      EXPECT_EQ(packages[0].display_version, "6.11.2");
      EXPECT_EQ(packages[0].url, "http://www.test.com:6666/package/nodejs/6.11.2/x86/node-v6.11.2-linux-x86.zip");
      EXPECT_EQ(packages[0].MD5, "test");
      EXPECT_EQ(packages[0].publisher, "test");
      EXPECT_EQ(packages[0].arch, "x86");
    }
#endif
  }
}

TEST(TestPackageManager, UninstallAction) {
  std::string response = "";
#ifdef _WIN32
  response = "[{\"packageId\":\"1\",\"name\":\"python3\",\
      \"url\":\"http://www.test.com:6666/package/python3/3.6.1/x86/python-3.6.1.zip\",\
      \"md5\":\"39192e116dce49bbd05efeced7924bae\",\"version\":\"3.6.1\",\
      \"publisher\":\"Python Software Foundation\",\"arch\":\"x86\"}]";
#else
  response = "[{\"packageId\":\"1\",\"name\":\"nodejs\",\
      \"url\":\"http://www.test.com:6666/package/nodejs/6.11.2/x86/node-v6.11.2-linux-x86.zip\",\
      \"md5\":\"test\",\"version\":\"6.11.2\",\
      \"publisher\":\"test\",\"arch\":\"x86\"}]";
#endif

  alyun_assist_installer::PackageManager package_mgr;
  vector<alyun_assist_installer::PackageInfo> package_infos =
      package_mgr.ParseResponseString(response);

  EXPECT_EQ(package_infos.size() > 0, true);
  if (package_infos.size() > 0)
  {
    package_mgr.UninstallAction(package_infos[0]);
    alyun_assist_installer::DBManager* db_manager =
        new alyun_assist_installer::DBManager();
#ifdef _WIN32
	std::vector<alyun_assist_installer::PackageInfo> packages =
        db_manager->GetPackageInfos("python3", "3.6.1");
    delete db_manager;
    EXPECT_EQ(packages.size() == 0, true);
#else
    std::vector<alyun_assist_installer::PackageInfo> packages =
        db_manager->GetPackageInfos("nodejs", "6.11.2");
    delete db_manager;
    EXPECT_EQ(packages.size() == 0, true);
#endif
  }
}
