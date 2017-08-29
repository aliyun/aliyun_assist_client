// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "./updatechecker.h"

#include <ctime>
#include <vector>
#include <string>
#include <cstdlib>
#include <algorithm>

#include "./appcast.h"
#include "utils/http_request.h"
#include "utils/OsVersion.h"
#include "utils/FileVersion.h"
#include "utils/ProcessUtil.h"
#include "utils/AssistPath.h"
#include "utils/DirIterator.h"
#include "utils/CheckNet.h"
#include "utils/FileUtil.h"
#include "utils/service_provide.h"
#include "zip/zip.h"
#include "jsoncpp/json.h"
#include "utils/Log.h"
#include "md5/md5.h"

#include "../VersionInfo.h"

namespace alyun_assist_update {

UpdateProcess::UpdateProcess(Appcast update_info) {
  update_info_ = update_info;
}

std::string UpdateProcess::get_request_string() {
  Json::Value jsonRoot;
  try {
#ifdef _WIN32
    jsonRoot["os"] = "windows";
#else
    jsonRoot["os"] = "linux";
#endif
    jsonRoot["os_version"] = OsVersion::GetVersion();
    jsonRoot["app_id"] = "aliyun assistant";
    jsonRoot["app_version"] = FILE_VERSION_RESOURCE_STR;
  } catch (...) {
    Log::Error("get_request_string failed");
  }
  return jsonRoot.toStyledString();
}

#if defined(TEST_MODE)
bool UpdateProcess::test_parse_response_string(std::string response) {
  return parse_response_string(response);
}
#endif

bool UpdateProcess::parse_response_string(std::string response) {
  Json::Value jsonRoot;
  Json::Reader reader;
  try {
    if (!reader.parse(response, jsonRoot)) {
      Log::Error("invalid json format");
      return false;
    }

    update_info_.need_update = jsonRoot["need_update"].asInt();
    if (update_info_.need_update == 0) {
      Log::Info("not need update");
      return false;
    }
    update_info_.flag = jsonRoot["flag"].asInt();

    Json::Value url_info = jsonRoot["update_info"];
    update_info_.download_url = url_info["url"].asString();
    update_info_.md5 = url_info["md5"].asString();
    update_info_.file_name = url_info["file_name"].asString();
    Log::Info("url:%s", update_info_.download_url.c_str());
  } catch(...) {
    Log::Error("update check json is invalid");
  }
  return true;
}

bool UpdateProcess::CheckUpdate() {
  std::string json = get_request_string();
  std::string response;
  if (HostChooser::m_HostSelect.empty()) {
    return false;
  }
  std::string url = ServiceProvide::GetUpdateService();
  HttpRequest::http_request_post(url, json, response);
  Log::Info("check update response:%s", response.c_str());
  return parse_response_string(response);
}

bool UpdateProcess::Download(const std::string url, const std::string path) {
  Log::Info("begin download file");
  return HttpRequest::download_file(url, path);
}

bool UpdateProcess::UnZip(const std::string file_path, const std::string dir) {
  Log::Info("begin unzip file");
  return 0 == zip_extract(file_path.c_str(), dir.c_str(), nullptr, nullptr);
}


void UninstallService() {
  AssistPath path_service("");
  std::string exe_path = path_service.GetCurrDir();
  exe_path += FileUtils::separator();
  exe_path.append("uninstall.bat");
  ProcessUtils::runSync(exe_path, "");
}

bool UpdateProcess::InstallFiles(const std::string src_dir,
    const std::string des_dir) {
  Log::Info("begin install files");
  InstallFilesRecursive(src_dir, des_dir);
  return true;
}

bool UpdateProcess::InstallFile(std::string src_path, std::string des_path) {
  // create the target directory if it does not exist
  std::string dest_dir = FileUtils::dirname(des_path.c_str());
  if (!FileUtils::fileExists(dest_dir.c_str())) {
    FileUtils::mkpath(dest_dir.c_str());
  }

  if (FileUtils::fileExists(src_path.c_str())) {
    FileUtils::copyFile(src_path.c_str(), des_path.c_str());
  } else {
    return false;
  }

  return true;
}

bool UpdateProcess::InstallFilesRecursive(std::string src_dir,
    std::string dst_dir) {
  DirIterator dir(src_dir.c_str());
  while (dir.next()) {
    std::string name = dir.fileName();
    if (name != "." && name != "..") {
      if (dir.isDir()) {
        std::string src_dir_new, dst_dir_new;
        src_dir_new = src_dir;
        src_dir_new += FileUtils::separator();
        src_dir_new += name;

        dst_dir_new = dst_dir;
        dst_dir_new += FileUtils::separator();
        dst_dir_new += name;
        InstallFilesRecursive(src_dir_new, dst_dir_new);
      } else {
        std::string dst_file_path;
        dst_file_path += dst_dir;
        dst_file_path += FileUtils::separator();
        dst_file_path += name;
        InstallFile(dir.filePath(), dst_file_path);
        // install service after update.
#if !defined(TEST_MODE)
#if defined(_WIN32)
        if (!name.compare("install.bat")) {
          ProcessUtils::runSync(dst_file_path, "");
        }
#else
        char buf[1024] = { 0 };
        sprintf(buf, "ln -sf %s /usr/sbin/aliyun-service", dst_file_path.c_str());
        if (!name.compare("aliyun-service")) {
          ProcessUtils::runSync(buf, "");
        }
#endif
#endif
      }
    }
  }
  return true;
}

bool UpdateProcess::CheckMd5(const std::string path,
    const std::string md5_string) {
  /*std::string content;
  FileUtils::ReadFileToString(path, content);
  md5 md5_service(content);
  std::string file_md5 = md5_service.Md5();
  if (md5_string.compare(file_md5) == 0) {
    return true;
  }
  else {
    return false;
  }*/
  return true;
}

bool UpdateProcess::RemoveOldVersion(std::string dir) {
  if (dir.find("assist") != std::string::npos) {
    Log::Info("begin remove old version");
    FileUtils::rmdirRecursive(dir.c_str());
    return true;
  }
  return false;
}

}  // namespace alyun_assist_update
