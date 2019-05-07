// Copyright (c) 2017-2018 Alibaba Group Holding Limited.
#include "./updatechecker.h"

#include <ctime>
#include <vector>
#include <string>
#include <cstdlib>
#include <algorithm>
#include <regex>

#include "./appcast.h"
#include "utils/http_request.h"
#include "utils/OsVersion.h"
#include "utils/FileVersion.h"
#include "utils/AssistPath.h"
#include "utils/DirIterator.h"
#include "utils/host_finder.h"
#include "utils/FileUtil.h"
#include "utils/process.h"
#include "utils/service_provide.h"
#include "utils/VersionComparator.h"
#include "zip/zip.h"
#include "json11/json11.h"
#include "utils/Log.h"
#include "md5/md5.h"

#include "../VersionInfo.h"

namespace alyun_assist_update {

UpdateProcess::UpdateProcess(Appcast update_info) {
  update_info_ = update_info;
}

std::string UpdateProcess::get_request_string() {
 
	json11::Json  json = json11::Json::object {
#ifdef _WIN32
		{"os","windows"},
#else
		{"os", "linux"},
#endif
		{ "os_version", OsVersion::GetVersion() },
	    { "app_id", "aliyun assistant" },
	    { "app_version",FILE_VERSION_RESOURCE_STR }
	};

	return json.dump();
 
}

#if defined(TEST_MODE)
bool UpdateProcess::test_parse_response_string(std::string response) {
  return parse_response_string(response);
}
#endif

bool UpdateProcess::parse_response_string(std::string response) {
  
  try {
	  string errinfo;
	  auto json = json11::Json::parse(response, errinfo);
	  if (errinfo != "") {
		  Log::Error("invalid json format");
		  return false;
	  }

    update_info_.need_update = json["need_update"].int_value();
    if (update_info_.need_update == 0) {
      Log::Info("not need update");
      return false;
    }
    update_info_.flag = json["flag"].int_value();

    update_info_.download_url = json["update_info"]["url"].string_value();
    update_info_.md5 = json["update_info"]["md5"].string_value();
    update_info_.file_name = json["update_info"]["file_name"].string_value();
    Log::Info("url:%s", update_info_.download_url.c_str());

  } catch(...) {
    Log::Error("update check json is invalid");
  }
  return true;
}

bool UpdateProcess::CheckUpdate() {
  std::string json = get_request_string();
  std::string response;
  if ( HostFinder::getServerHost().empty() ) {
    return false;
  }
  std::string url = ServiceProvide::GetUpdateService();
  HttpRequest::https_request_post(url, json, response);
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
  Process(exe_path).syncRun();
}

bool UpdateProcess::InstallFiles(const std::string src_dir,
    const std::string des_dir) {
  Log::Info("begin install files");
  InstallFilesRecursive(src_dir, des_dir);
  update_script();
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

bool UpdateProcess::update_script() {

#if !defined(TEST_MODE)
  Log::Info("install update script, path:%s", script_dir.c_str());
#if defined(_WIN32)
  Process(script_dir).syncRun();

#else
  std::string content;
  FileUtils::readFile(script_dir, content);
  Process(content).syncRun();
#endif
#endif
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
          script_dir = dst_file_path;
        }
#else
        if (!name.compare("update_install")) {
          script_dir = dst_file_path;
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
  
  std::string origal = md5_string;
  transform(origal.begin(), origal.end(), origal.begin(), (int(*)(int))tolower);

  std::string file_md5 = md5file( path.c_str() );
  transform(file_md5.begin(), file_md5.end(), file_md5.begin(), (int(*)(int))tolower);

  if (origal.compare(file_md5) == 0) {
    return true;
  }
  else {
    return false;
  }
}

bool UpdateProcess::RemoveOldVersion(std::string dir) {
  Log::Info("Remove old version from %s", dir.c_str());
  if (dir.empty()) {
    Log::Error("Install dir is empty");
    return false;
  }

  std::string current_version = FILE_VERSION_RESOURCE_STR;
  DirIterator it_dir(dir.c_str());
  while (it_dir.next()) {
    std::string name = it_dir.fileName();
    if (name == "." || name == "..")
      continue;

    if (!it_dir.isDir())
      continue;

    //Filter the sub dir which name is not like 1.0.0.130
    regex reg("\\d+(.\\d+){3}");
    smatch str_match;
    if (!regex_match(name, str_match, reg))
      continue;

    if (VersionComparator::CompareVersions(name, current_version) < 0) {
      std::string outdated_dir = dir + FileUtils::separator() + name;
      Log::Info("Remove old version: %s", outdated_dir.c_str());
      FileUtils::rmdirRecursive(outdated_dir.c_str());
    }
  }

  return true;
}

}  // namespace alyun_assist_update
