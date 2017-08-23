// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "gtest/gtest.h"

#include <string>
#include <algorithm>

#include "utils/http_request.h"
#include "utils/AssistPath.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"
#include "curl/curl.h"
#include "utils/CheckNet.h"
#include "utils/OsVersion.h"
#include "jsoncpp/json.h"
#include "md5/md5.h"
#include "../update/update_check/updatechecker.h"
#include "../update/update_check/appcast.h"

TEST(TestUpdate, Unzip) {
  std::string cur_path, cur_dir;
  AssistPath path_service("");
  cur_dir = path_service.GetCurrDir();
  cur_path += cur_dir;
  cur_path += FileUtils::separator();
  cur_path += "test.zip";
  alyun_assist_update::Appcast update_info;
  alyun_assist_update::UpdateProcess process(update_info);
  bool ret = process.UnZip(cur_path, cur_dir);
  EXPECT_EQ(true, ret);
}
//
TEST(TestUpdate, InstallFiles) {
  std::string cur_dir;
  AssistPath path_service("");
  cur_dir = path_service.GetCurrDir();
  cur_dir += FileUtils::separator();
  cur_dir += "config";
  std::string dest_dir = cur_dir;
  dest_dir += FileUtils::separator();
  dest_dir += "testInstallFiles";
  alyun_assist_update::Appcast update_info;
  alyun_assist_update::UpdateProcess process(update_info);
  bool ret = process.InstallFiles(cur_dir, dest_dir);
  EXPECT_EQ(true, ret);
}

TEST(TestUpdate, CheckMd5) {
  std::string cur_path;
  AssistPath path_service("");
  cur_path = path_service.GetCurrDir();
  cur_path += FileUtils::separator();
  cur_path += "test.zip";
  std::string content;
  FileUtils::ReadFileToString(cur_path, content);
  md5 md5_service(content);
  std::string file_md5 = md5_service.Md5();
  bool ret = false;
  std::string md5_string("36b50774d96789879d7443aba7d9514a");
  if (md5_string.compare(file_md5) == 0) {
    ret = true;
  }
  EXPECT_EQ(true, ret);
}

//TEST(TestUpdate, DownloadFile) {
//  std::string url("http://localhost:8080/test.zip");
//  std::string path;
//  AssistPath path_service("");
//  path = path_service.GetCurrDir();
//  path += FileUtils::separator();
//  path += "test_download.zip";
//  HttpRequest::download_file(url, path);
//  bool ret = FileUtils::fileExists(path.c_str);
//  EXPECT_EQ(true, ret);
//}

TEST(TestUpdate, CheckUpdate) {
  std::string mocked_string("{\"need_update\":1}");
  alyun_assist_update::Appcast update_info;
  alyun_assist_update::UpdateProcess process(update_info);
  bool ret = process.test_parse_response_string(mocked_string);
  EXPECT_EQ(true, ret);
}

