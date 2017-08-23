// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#ifndef CLIENT_PACKAGE_INSTALLER_PACKAGEINFO_H_
#define CLIENT_PACKAGE_INSTALLER_PACKAGEINFO_H_
#include <string>

namespace alyun_assist_installer {

struct PackageInfo {
  std::string package_id;
  std::string display_name;
  std::string display_version;
  std::string new_version;
  std::string publisher;
  std::string install_date;
  std::string url;
  std::string MD5;
  std::string arch;
};

}  // namespace alyun_assist_installer

#endif  // CLIENT_PACKAGE_INSTALLER_PACKAGEINFO_H_
