// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_UPDATE_UPDATE_CHECK_UPDATECHECKER_H_
#define CLIENT_UPDATE_UPDATE_CHECK_UPDATECHECKER_H_

#include <string>
#include "./appcast.h"

namespace alyun_assist_update {

struct Appcast;
class UpdateProcess {
 public:
  explicit UpdateProcess(Appcast update_info);
  bool CheckUpdate();
  bool Download(const std::string url, const std::string path);
  bool UnZip(const std::string file_path, const std::string dir);
  bool CheckMd5(const std::string path, const std::string md5_string);
  bool InstallFiles(const std::string src_dir, const std::string des_dir);
  static bool RemoveOldVersion(const std::string dir);
#if defined(TEST_MODE)
  bool test_parse_response_string(std::string response);
#endif
  Appcast GetUpdateInfo() { return update_info_; }
  void SetUpdateInfo(Appcast update_info) { update_info_ = update_info; }
 private:
  std::string get_request_string();
  bool update_script();
  bool parse_response_string(std::string response);
  bool InstallFilesRecursive(std::string src_dir, std::string dst_dir);
  bool InstallFile(std::string src_path, std::string des_path);
  Appcast update_info_;
  std::string script_dir;
};

}  // namespace alyun_assist_update

#endif  // CLIENT_UPDATE_UPDATE_CHECK_UPDATECHECKER_H_
