// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_UPDATE_UPDATE_CHECK_APPCAST_H_
#define CLIENT_UPDATE_UPDATE_CHECK_APPCAST_H_
#include <string>
#include <vector>

namespace alyun_assist_update {
struct Cmd {
  std::string path;
  std::string params;
};
struct Appcast {
  /// App version fields
  std::string version;
  int need_update;
  int flag;

  /// URL of the update
  std::string download_url;
  std::string md5;
  std::string file_name;

  // Operating system
  std::string OS;
 
  Appcast() {
	flag = 0;
	need_update = 0;
  }
};

}  // namespace alyun_assist_update

#endif  // CLIENT_UPDATE_UPDATE_CHECK_APPCAST_H_
