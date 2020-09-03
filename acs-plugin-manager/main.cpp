// Copyright (c) 2017-2018 Alibaba Group Holding Limited.

#include <string>
#include "acs_plugin_manager.h"
#include "optparse/OptionParser.h"
#include "utils/process.h"
#include "utils/Log.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/host_finder.h"
#include "../VersionInfo.h"
#include "bprinter/table_printer.h"

#ifdef _WIN32
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#else
#include <sys/file.h>
#include <unistd.h>
#endif



using optparse::OptionParser;

OptionParser& initParser() {
  static OptionParser parser = OptionParser().description("Aliyun Assist Copyright (c) 2017-2018 Alibaba Group Holding Limited");

  parser.add_option("-v", "--version")
    .action("store_true")
    .help("show version and exit");
    
  parser.add_option("-V", "--verbose")
    .action("store_true")
    .help("show version and exit");

  parser.add_option("-l", "--list")
      .dest("list")
      .action("store_true")
      .help("--list show all plugins,--list --local means only local installed plugins");

  parser.add_option("-L", "--local")
    .dest("local")
    .action("store_true");

  parser.add_option("-f", "--verify")
    .dest("verify")
    .action("store_true")
    .help("verify plugin, --verify --url <> --params <>");

  parser.add_option("-P", "--plugin")
      .dest("plugin")
      .action("store");

    parser.add_option("-p", "--params")
      .dest("params")
      .action("store");

    parser.add_option("-u", "--url")
      .dest("url")
      .action("store");

    parser.add_option("-s", "--separator")
      .dest("separator")
      .action("store");

    parser.add_option("-e", "--exec")
      .action("store_true")
      .help("exec plugin, --exec --pluginid <> --params <>");

    return parser;
}

bool process_singleton() {
#ifdef _WIN32
	CreateMutex(NULL, FALSE, L"alyun_assist_plugin");
	if (GetLastError() == ERROR_ALREADY_EXISTS) {
		return false;
	}
	else {
		return true;
	}

#else
	static int lockfd = open("/var/tmp/alyun_assist_plugin.lock", O_CREAT | O_RDWR, 0644);
	if (-1 == lockfd) {
		Log::Error("Fail to open lock file. Error: %s\n", strerror(errno));
		return false;
	}

	if (0 == flock(lockfd, LOCK_EX | LOCK_NB)) {
		atexit([] {
			close(lockfd);
		});
		return true;
	}

	close(lockfd);
	return false;
#endif
	}

//////////////////////////////////////////////////////////////
void check_endpoint() {
  if ( HostFinder::getServerHost().empty() ) {
    printf("cound not find a endpoint to connect server");
    exit(1);
  }
}

/////////////////////////////////////////////////////////////

int main(int argc, char *argv[]) {

  AssistPath path_service("");
  std::string log_path = path_service.GetLogPath();
  log_path += FileUtils::separator();
  log_path += "acs_plugin_manager.log";
  Log::Initialise(log_path);
  
  OptionParser& parser = initParser();
  optparse::Values options = parser.parse_args(argc, argv);

  if (options.is_set("version")) {
    printf("%s\n", FILE_VERSION_RESOURCE_STR);
    return 0;
  }

  if ( !process_singleton() ) {
    if (options.is_set("verbose"))
      printf("exit by another plugin process is running");
    Log::Error("exit by another plugin process is running");
    return 1;
  }

  bool verbose = false;
  if(options.is_set("verbose"))
    verbose = true;
  acs::PluginManager plugin_mgr(verbose);


  if (options.is_set("list")) {
    check_endpoint();
    if(options.is_set("local")) {
      return plugin_mgr.ListLocal();
    } else {
      std::string pluginName = options.get("plugin");
      return plugin_mgr.List(pluginName);
    }

    return 0;
  }

  if (options.is_set("exec")) {
    check_endpoint();
    std::string pluginName = options.get("plugin");
    std::string params = options.get("params");
    std::string separator = options.get("separator");
    return plugin_mgr.installPlugin(pluginName, params, separator);
  }

  if (options.is_set("verify")) {
    check_endpoint();
    std::string url = options.get("url");
    std::string params = options.get("params");
    std::string separator = options.get("separator");
    return plugin_mgr.verifyPlugin(url, params, separator);
  }


  parser.print_help();
}

