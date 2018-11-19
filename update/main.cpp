// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include "windows.h"
#endif
#include <string>

#include "update_check/updatechecker.h"
#include "optparse/OptionParser.h"
#include "utils/SubProcess.h"
#include "utils/Log.h"
#include "utils/AssistPath.h"
#include "utils/CheckNet.h"
#include "utils/FileUtil.h"
#include "utils/ProcessUtil.h"
#include "update_check/appcast.h"
#include "curl/curl.h"
#include "../VersionInfo.h"
#include "utils/dump.h"

using optparse::OptionParser;

OptionParser& initParser() {
    static OptionParser parser = OptionParser().description("Aliyun Assist Copyright (c) 2017-2018 Alibaba Group Holding Limited");

  parser.add_option("-v", "--version")
      .dest("version")
      .action("store_true")
      .help("show version and exit");

  parser.add_option("-u", "--check_update")
      .action("store_true")
      .dest("check_update")
      .help("Check and update if necessary");

  parser.add_option("-f", "--force_update")
      .action("store_true")
      .dest("force_update");

  parser.add_option("-u", "--url")
      .dest("url")
      .action("store");
  return parser;
}

bool process_singleton() {
#ifdef _WIN32
  if (NULL == ::CreateMutex(NULL, FALSE, L"alyun_assist_update")) {
    if (GetLastError() == ERROR_ALREADY_EXISTS) {
      return false;
    }
  }
#else
  return ProcessUtils::is_single_proc_inst_running("aliyun_assist_update");
#endif
  return true;
}

int main(int argc, char *argv[]) {
#if defined(_WIN32)
  SetDllDirectory(TEXT(""));
  DumpService::InitMinDump("aliyun update");
#endif
  AssistPath path_service("");
  std::string log_path = path_service.GetLogPath();
  log_path += FileUtils::separator();
  log_path += "aliyun_assist_update.log";
  Log::Initialise(log_path);
  Log::Info("process begin...");

  OptionParser& parser = initParser();
  optparse::Values options = parser.parse_args(argc, argv);

  if (options.is_set("version")) {
    printf("%s\n", FILE_VERSION_RESOURCE_STR);
    return 0;
  }

  if (options.is_set("check_update")) {
    std::string cur_dir = path_service.GetCurrDir();
    std::string test_file = cur_dir + FileUtils::separator() + "no_update";
    if (FileUtils::fileExists(test_file.c_str())) {
      Log::Info("in  test mode no need update");
      return 0;
    }
    if (!process_singleton()) {
      Log::Error("exit by another update process is running");
      return -1;
    }
    curl_global_init(CURL_GLOBAL_ALL);
    HostChooser host_choose;
    bool found = host_choose.Init(path_service.GetConfigPath());
    if (!found) {
      Log::Error("could not find a match region host");
    }
    alyun_assist_update::Appcast update_info;
    memset(&update_info, 0, sizeof (alyun_assist_update::Appcast));
    alyun_assist_update::UpdateProcess process(update_info);
    bool need_update = process.CheckUpdate();
    // In test mode, we use download url pass form command line.
    std::string test_force_update_file = cur_dir + FileUtils::separator() + ".." + FileUtils::separator() + "force_update";
    if(options.is_set("force_update") || FileUtils::fileExists(test_force_update_file.c_str())) {
      need_update = true;
      alyun_assist_update::Appcast cast;
      cast.need_update = 1;
      cast.flag = 0;
      std::string url =  options.get("url");
      if(url.empty()) {
        url = "https://repo-aliyun-assist.oss-cn-beijing.aliyuncs.com/download/update.zip";
      }
      cast.download_url = url;
      process.SetUpdateInfo(cast);
    }
    if (need_update) {
      update_info = process.GetUpdateInfo();
      std::string tmp_path, tmp_dir, unzip_dest_dir;
      path_service.GetTmpPath(tmp_dir);
      tmp_path += tmp_dir;
      tmp_path += FileUtils::separator();
      std::string file_name, file_dir;
      file_name += "aliyun_assist_";
      file_name += update_info.md5;
      file_dir = file_name;
      file_name += ".zip";
      update_info.file_name = file_name;
      tmp_path.append(file_name);

      bool download_ret = process.Download(update_info.download_url, tmp_path);
      if (!download_ret) {
        Log::Error("download zip failed,url:%s",
            update_info.download_url.c_str());
        return 0;
      }
      if (!process.CheckMd5(tmp_path, update_info.md5)) {
        Log::Error("check file md5 failed");
        return 0;
      }
      unzip_dest_dir = tmp_dir;
      unzip_dest_dir += FileUtils::separator();
      unzip_dest_dir += file_dir;
      if (!process.UnZip(tmp_path, unzip_dest_dir)) {
        Log::Error("unzip file failed");
        return 0;
      }

      std::string cur_dir, install_dir;
      cur_dir = path_service.GetCurrDir();
      Log::Info("current dir :%s", cur_dir.c_str());
      char ctemp[1024] = { 0 };
      strncpy(ctemp, cur_dir.c_str(), cur_dir.length());
      char *pPath = strrchr(ctemp, FileUtils::separator());
      if(!pPath) {
        Log::Error("install path errors");
        return 0;
      }
      *pPath = '\0';
      install_dir = string(ctemp);
      Log::Info("install from  %s to %s",
          unzip_dest_dir.c_str(), install_dir.c_str());
      if(!process.InstallFiles(unzip_dest_dir, install_dir)) {
        Log::Error("install files failed");
        return 0;
      }
    }
    curl_global_cleanup();
    return 0;
  }

  parser.print_help();
}

