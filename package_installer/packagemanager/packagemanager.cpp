// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include "./packagemanager.h"
#include <string>
#include <vector>
#include <algorithm>
#include "./packageinfo.h"
#include "utils/AssistPath.h"
#include "utils/OsVersion.h"
#include "utils/FileVersion.h"
#include "utils/CheckNet.h"
#include "utils/http_request.h"
#include "utils/Log.h"
#include "utils/FileUtil.h"
#include "jsoncpp/json.h"
#include "zip/zip.h"
#include "md5/md5.h"

namespace alyun_assist_installer {
namespace {
// String characters classification. Valid components of version numbers
// are numbers, period or string fragments ("beta" etc.).
enum CharType {
  Type_Number,
  Type_Period,
  Type_String
};

CharType ClassifyChar(char c) {
  if (c == '.')
    return Type_Period;
  else if (c >= '0' && c <= '9')
    return Type_Number;
  else
    return Type_String;
}

// Split version string into individual components. A component is continuous
// run of characters with the same classification. For example, "1.20rc3" would
// be split into ["1",".","20","rc","3"].
vector<string> SplitVersionString(const string& version) {
  vector<string> list;

  if (version.empty())
    return list;  // nothing to do here

  string s;
  const size_t len = version.length();

  s = version[0];
  CharType prevType = ClassifyChar(version[0]);

  for (size_t i = 1; i < len; i++) {
    const char c = version[i];
    const CharType newType = ClassifyChar(c);

    if (prevType != newType || prevType == Type_Period) {
      // We reached a new segment. Period gets special treatment,
      // because "." always delimiters components in version strings
      // (and so ".." means there's empty component value).
      list.push_back(s);
      s = c;
    } else {
      // Add character to current segment and continue.
      s += c;
    }

    prevType = newType;
  }

  // Don't forget to add the last part:
  list.push_back(s);

  return list;
}

}  // anonymous namespace

PackageManager::PackageManager() {
  db_manager = new DBManager();
}

PackageManager::~PackageManager() {
  delete db_manager;
}

int PackageManager::CompareVersions(const string& verA, const string& verB) {
  const vector<string> partsA = SplitVersionString(verA);
  const vector<string> partsB = SplitVersionString(verB);

  // Compare common length of both version strings.
  const size_t n = min(partsA.size(), partsB.size());
  for (size_t i = 0; i < n; i++) {
    const string& a = partsA[i];
    const string& b = partsB[i];

    const CharType typeA = ClassifyChar(a[0]);
    const CharType typeB = ClassifyChar(b[0]);

    if (typeA == typeB) {
      if (typeA == Type_String) {
        int result = a.compare(b);
        if (result != 0)
          return result;
      } else if (typeA == Type_Number) {
        const int intA = atoi(a.c_str());
        const int intB = atoi(b.c_str());
        if (intA > intB)
          return 1;
        else if (intA < intB)
          return -1;
      }
    } else {  // components of different types
      if (typeA != Type_String && typeB == Type_String) {
        // 1.2.0 > 1.2rc1
        return 1;
      } else if (typeA == Type_String && typeB != Type_String) {
        // 1.2rc1 < 1.2.0
        return -1;
      } else {
        // One is a number and the other is a period. The period
        // is invalid.
        return (typeA == Type_Number) ? 1 : -1;
      }
    }
  }

  // The versions are equal up to the point where they both still have
  // parts. Lets check to see if one is larger than the other.
  if (partsA.size() == partsB.size())
    return 0;  // the two strings are identical

              // Lets get the next part of the larger version string
              // Note that 'n' already holds the index of the part we want.

  int shorterResult, longerResult;
  CharType missingPartType;  // ('missing' as in "missing in shorter version")

  if (partsA.size() > partsB.size()) {
    missingPartType = ClassifyChar(partsA[n][0]);
    shorterResult = -1;
    longerResult = 1;
  } else {
    missingPartType = ClassifyChar(partsB[n][0]);
    shorterResult = 1;
    longerResult = -1;
  }

  if (missingPartType == Type_String) {
    // 1.5 > 1.5b3
    return shorterResult;
  } else {
    // 1.5.1 > 1.5
    return longerResult;
  }
}

void PackageManager::List(const std::string& package_name) {
  Log::Info("Enter list, package_name: %s", package_name);
  vector<PackageInfo> package_infos = GetPackageInfo(package_name);

  if (package_infos.empty()) {
    if (package_name.empty()) {
      Log::Error("There no package in the software store");
      printf("There no package in the software store\n");
    } else {
      Log::Info("There no package named %s in the software store",
          package_name.c_str());
      printf("There no package named %s in the software store\n",
          package_name.c_str());
    }
  } else {
    printf("package_id\tname\tversion\tarch\tpublisher\n");
    for (size_t i = 0; i < package_infos.size(); ++i) {
      printf("%s\t%s\t%s\t%s\t%s\n", package_infos[i].package_id.c_str(),
        package_infos[i].display_name.c_str(),
        package_infos[i].display_version.c_str(),
        package_infos[i].arch.c_str(),
        package_infos[i].publisher.c_str());
    }
  }
}

void PackageManager::Local(const std::string& package_name) {
  Log::Info("Enter Local, package_name: %s", package_name);
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, false);

  if (!package_infos.empty()) {
    if (package_name.empty()) {
      Log::Info("There no package in the local");
      printf("There no package in the local\n");
    } else {
      Log::Info("There no package named %s in the local", package_name);
      printf("There no package named %s in the local\n", package_name.c_str());
    }
  } else {
    printf("name\tversion\tpublisher\tinstall data\n");
    for (size_t i = 0; i < package_infos.size(); ++i) {
      printf("%s\t%s\t%s\t%s\t%s\t%s\n", package_infos[i].package_id.c_str(),
        package_infos[i].display_name.c_str(),
        package_infos[i].display_version.c_str(),
        package_infos[i].arch.c_str(),
        package_infos[i].publisher.c_str(),
        package_infos[i].install_date.c_str());
    }
  }
}

void PackageManager::Latest(const std::string& package_name) {
  Log::Info("Enter Latest, package_name: %s", package_name);
  // query the package in the local
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, false);

  vector<PackageInfo> new_packages;
  for (size_t i = 0; i < package_infos.size(); ++i) {
    package_infos[i].new_version = package_infos[i].display_version;
    vector<PackageInfo> packages =
        GetPackageInfo(package_infos[i].display_name);
    for (size_t j = 0; j < packages.size(); j++) {
      if ((package_infos[i].display_name == packages[j].display_name) &&
          (package_infos[i].arch == packages[j].arch)) {
        // compare the version of the local package and remote package
        if (CompareVersions(packages[j].display_version,
            package_infos[i].new_version) > 0) {
          package_infos[i].new_version = packages[j].display_version;
        }
      }
    }
  }

  for (size_t i = 0; i < package_infos.size(); ++i) {
    if (package_infos[i].new_version != package_infos[i].display_version) {
      new_packages.push_back(package_infos[i]);
    }
  }

  if (!new_packages.empty()) {
    printf("name\tversion\tnewversion\tarch\tpublisher\n");
    for (size_t i = 0; i < new_packages.size(); ++i) {
      printf("%s\t%s\t%s\t%s\t%s\n", new_packages[i].display_name.c_str(),
          new_packages[i].display_version.c_str(),
          new_packages[i].new_version.c_str(),
          new_packages[i].arch.c_str(),
          new_packages[i].publisher.c_str());
    }
  }
}

void PackageManager::Install(const std::string& package_name,
    const std::string& package_version,
    const std::string& arch) {
  Log::Info("Enter Install, package_name: %s, package_version: %s,arch: %s",
      package_name, package_version, arch);

  // If the package_version is empty, fuzzy query the package_name
  if (package_version.empty()) {
    vector<PackageInfo> package_infos = GetPackageInfo(package_name);
    if (!package_infos.empty()) {
      printf("package_id\tname\tversion\tarch\tpublisher\n");
      for (size_t i = 0; i < package_infos.size(); ++i) {
        printf("%s\t%s\t%s\t%s\t%s\n", package_infos[i].package_id.c_str(),
            package_infos[i].display_name.c_str(),
            package_infos[i].display_version.c_str(),
            package_infos[i].arch.c_str(),
            package_infos[i].publisher.c_str());
      }
    }

    // If there are many packages whose name include package_name,
    // ask user to input the package_id
    printf("Please input the package_id you want to install.\n");
    char package_id[100];
    scanf_s("%s", package_id, 99);
    for (size_t i = 0; i < package_infos.size(); ++i) {
      if (package_infos[i].package_id == package_id) {
        CheckInstall(package_infos[i]);
        break;
      }
    }
  } else {
    // If the package_version is not empty, accurate query the package_name
    vector<PackageInfo> package_infos =
        GetPackageInfo(package_name, package_version, arch);
    if (package_infos.empty()) {
      printf("There is no such package.\n");
    } else {
      CheckInstall(package_infos[0]);
    }
  }
}

void PackageManager::Uninstall(const std::string& package_name) {
  Log::Info("Enter Uninstall, package_name: %s", package_name);
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, true);
  if (!package_infos.empty()) {
    UninstallAction(package_infos[0]);
  } else {
    Log::Info("Uninstall failed, no package named %s", package_name);
    printf("There is no such package.\n");
    return;
  }
}

void PackageManager::Update(const std::string& package_name) {
  Log::Info("Enter Update, package_name: %s", package_name);
  vector<PackageInfo> package_infos =
      db_manager->GetPackageInfos(package_name, true);
  if (package_infos.empty()) {
    Log::Info("There no package named %s in the local", package_name);
    printf("There is no such package on this machine.\n");
    return;
  }

  std::string new_version = package_infos[0].display_version;
  vector<PackageInfo> packages = GetPackageInfo(package_infos[0].display_name);
  int index = -1;
  for (size_t i = 0; i < packages.size(); i++) {
    if ((package_infos[0].display_name == packages[i].display_name) &&
        (package_infos[0].arch == packages[i].arch)) {
      // compare the version of the local package and remote package
      if (CompareVersions(packages[i].display_version, new_version) > 0) {
        new_version = packages[i].display_version;
        index = i;
      }
    }
  }

  if (index == -1) {
    Log::Info("The package is latest. There is no need to update, %s",
        package_name);
    printf("The package is latest. There is no need to update.\n");
    return;
  }

  InstallAction(packages[index]);
  db_manager->Delete(package_infos[0].package_id);
}

void PackageManager::CheckInstall(const PackageInfo& package_info) {
  vector<PackageInfo> package_infos =
    db_manager->GetPackageInfosById(package_info.package_id);

  if (!package_infos.empty()) {
    printf("name\tversion\tpublisher\tinstall data\n");
    for (size_t i = 0; i < package_infos.size(); ++i) {
      printf("%s\t%s\t%s\t%s\n", package_infos[i].display_name.c_str(),
          package_infos[i].display_version.c_str(),
          package_infos[i].publisher.c_str(),
          package_infos[i].install_date.c_str());
      Log::Info("This package is already exist, %s.",
          package_infos[i].display_name);
      printf("This package is already exist.\n");
    }

    return;
  }

  InstallAction(package_info);
}

void PackageManager::InstallAction(const PackageInfo& package_info) {
  AssistPath path("");
  std::string userdata_path;
  path.GetDefaultUserDataDirectory(userdata_path);
  std::string file_path = userdata_path;
  std::string file_name = package_info.url.substr(
      package_info.url.find_last_of('/') + 1);
  file_path.append(file_name);
  bool download_ret = Download(package_info.url, file_path);
  if (!download_ret) {
    printf("Download this package failed, please try again later.\n");
    return;
  }

  if (!CheckMd5(file_path, package_info.MD5)) {
    printf("Check file md5 failed.\n");
    return;
  }

  bool unzip_ret = UnZip(file_path, userdata_path);
  if (!unzip_ret) {
    printf("Unzip this package failed, please try again later.\n");
    return;
  }

  std::string install_dir = userdata_path;
  install_dir.append(file_name.substr(0, file_name.find_last_of('.')));
  std::string install_file = install_dir;
  install_file.append("\\");
  install_file.append("install.bat");
  std::string cmd = install_file + " " + install_dir;

  if (system(cmd.c_str()) == 0) {
    vector<PackageInfo> package_infos;
    package_infos.push_back(package_info);
    db_manager->ReplaceInto(package_infos);
    remove(file_path.c_str());
  }
}

void PackageManager::UninstallAction(const PackageInfo& package_info) {
  AssistPath path("");
  std::string userdata_path;
  path.GetDefaultUserDataDirectory(userdata_path);
  std::string file_name = package_info.url.substr(
      package_info.url.find_last_of('/') + 1);
  std::string uninstall_dir = userdata_path;
  uninstall_dir.append(file_name.substr(0, file_name.find_last_of('.')));
  std::string uninstall_file = uninstall_dir;
  uninstall_file.append("\\");
  uninstall_file.append("uninstall.bat");
  std::string cmd = uninstall_file;

  if (system(cmd.c_str()) == 0) {
    db_manager->Delete(package_info.package_id);
  }
}

vector<PackageInfo> PackageManager::GetPackageInfo(
    const std::string& package_name,
    const std::string& package_version,
    const std::string& arch) {
  std::string response;
  std::string url = "http://" + HostChooser::m_HostSelect +
    "/luban/api/v1/repo/query_software?";
  if (!package_name.empty()) {
    url = url + "package_name=" + (package_name.empty() ? "*" : package_name);
    if (!package_version.empty()) {
      url = url + "&" + "package_version=" + package_version;
    }

    if (!arch.empty()) {
      url = url + "&" + "arch=" + arch;
    }
  }

#ifdef _WIN32
  url += "&os=windows";
#else
  url += "&os=linux";
#endif

  vector<PackageInfo> package_infos;
  bool ret = HttpRequest::http_request_post(url, "", response);
  if (ret) {
    package_infos = parse_response_string(response);
  } else {
    Log::Error("http request failed, url: %s, response:%s", url, response);
  }

  return package_infos;
}

std::string PackageManager::get_request_string(
    const std::string& package_name,
    const std::string& package_version) {
  Json::Value jsonRoot;
#ifdef _WIN32
  jsonRoot["os"] = "windows";
#else
  jsonRoot["os"] = "linux";
#endif
  jsonRoot["package_name"] = package_name;
  jsonRoot["package_version"] = package_version;
  return jsonRoot.toStyledString();
}

vector<PackageInfo> PackageManager::parse_response_string(
    std::string response) {
  Json::Value jsonRoot;
  Json::Reader reader;

  vector<PackageInfo> package_infos;
  if (!reader.parse(response, jsonRoot)) {
    Log::Error("invalid json format");
    return package_infos;
  }

  for (size_t i = 0; i < jsonRoot.size(); ++i) {
    PackageInfo package_info;
    package_info.package_id = jsonRoot[i]["packageId"].asString();
    package_info.url = jsonRoot[i]["url"].asString();
    package_info.MD5 = jsonRoot[i]["md5"].asString();
    package_info.display_name = jsonRoot[i]["name"].asString();
    package_info.display_version = jsonRoot[i]["version"].asString();
    package_info.publisher = jsonRoot[i]["publisher"].asString();
    package_info.arch = jsonRoot[i]["arch"].asString();
    package_infos.push_back(package_info);
  }

  return package_infos;
}

bool PackageManager::Download(const std::string& url,
    const std::string& path) {
  return HttpRequest::download_file(url, path);
  bool ret = HttpRequest::download_file(url, path);
  if (ret) {
    return true;
  } else {
    Log::Error("Download failed, url: %s", url);
    return false;
  }
}

bool PackageManager::CheckMd5(const std::string& path,
    const std::string& md5_string) {
  std::string content;
  FileUtils::ReadFileToString(path, content);
  md5 md5_service(content);
  std::string file_md5 = md5_service.Md5();
  if (md5_string.compare(file_md5) == 0) {
    return true;
  } else {
    Log::Error("UnZip failed, path: %s, file_md5: %s, md5_string: %s",
      path, file_md5, md5_string);
    return false;
  }
}

bool PackageManager::UnZip(const std::string& file_name,
    const std::string& dir) {
  int ret = zip_extract(file_name.c_str(), dir.c_str(), nullptr, nullptr);
  if (ret == 0) {
    return true;
  } else {
    Log::Error("UnZip failed, file name: %s", file_name);
    return false;
  }
}
}  // namespace alyun_assist_installer
