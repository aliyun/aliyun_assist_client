// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#define WIN32_LEAN_AND_MEAN
#include "optparse/OptionParser.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/Log.h"
#include "schedule_task.h"
#include "timer_manager.h"
#include "json11/json11.h"
#include "../VersionInfo.h"

#ifdef _WIN32
#include <windows.h>
#include "windows/service_app.h"
#else
#include "linux/service_app.h"
#endif
#include "utils/process.h"

using optparse::OptionParser;

namespace {

	void initLogger() {
		AssistPath  path_service;
		std::string log_path = path_service.GetLogPath();
		log_path += FileUtils::separator();
		log_path += "aliyun_assist_main.log";
		Log::Initialise(log_path);
	}


	OptionParser& initParser() {

		static OptionParser parser = OptionParser()
			.description("Aliyun Assist Copyright (c) 2017-2018 Alibaba Group Holding Limited");

		parser.add_option("-v", "--version")
			.dest("version")
			.action("store_true")
			.help("show version and exit");

		parser.add_option("-c", "--common")
			.action("store_true")
			.dest("common")
			.help("run as common");

		parser.add_option("-d", "--daemon")
			.action("store_true")
			.dest("daemon")
			.help("start as daemon");

#ifdef _WIN32
		parser.add_option("-s", "--service")
			.action("store_true")
			.dest("service")
			.help("start as daemon");

#endif
		return parser;
	}
}

int main( int argc, char *argv[] ) {

	initLogger();
	OptionParser& parser = initParser();
	optparse::Values options = parser.parse_args(argc, argv);
	
	if ( options.is_set("version") ) {
		printf("%s\n", FILE_VERSION_RESOURCE_STR);
		return 0;
	}
	if ( options.is_set("common") ) {
		Singleton<ServiceApp>::I().runCommon();
		return 0;
	}
	
	if ( options.is_set("daemon") ) {
		Singleton<ServiceApp>::I().becomeDeamon();
	}
	
	Singleton<ServiceApp>::I().runService();
	return 0;
}
