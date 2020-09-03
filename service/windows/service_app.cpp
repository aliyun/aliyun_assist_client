// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <thread>
#include <math.h>
#include "service_app.h"
#include "utils/singleton.h"
#include "./schedule_task.h"
#include "utils/AssistPath.h"
#include "../VersionInfo.h"
#include "json11/json11.h"
#include "utils/FileUtil.h"
#include "utils/process.h"
#include "utils/Log.h"
#include "timer_manager.h"
#include "../notifer_factory.h"
#include "utils/host_finder.h"
#include "utils/process.h"
#include "utils/OsUtil.h"
#include "utils/OsVersion.h"
#include "utils/service_provide.h"
#include "utils/TimeTool.h"
#include "utils/http_request.h"


using task_engine::TimerManager;


#define UPDATER_NAME	"aliyun_assist_update.exe"
#define UPDATER_COMMAND " --check_update"
#define DEFAULT_PING_INTERVAL_SECONDS 1800
#define PING_INTERVAL_SECONDS_MIN 30

extern function<void(char*)>    wskt_callback;

void  ServiceApp::onStart(DWORD argc, TCHAR* argv[]) {
	HostFinder::setStopPolling(false);
	m_updateFinish = false;
	Singleton<TimerManager>::I().start();
  AssistPath path;
  path.SetCurrentEnvPath();

	std::thread worker( [this] {
  wskt_callback = [this](const char* msg) {
		onCommand(msg);
  };


	Singleton<NotiferFactory>::I().init([this](const char* msg) {
		onCommand(msg);
	});


    int retryCount = 0;
    while ( HostFinder::getServerHost().empty() ) {
      int second = int(pow(2, retryCount));
      Sleep(second * 1000);
      if (retryCount < 10)
        retryCount++;
    }

    m_pingTimer = Singleton<TimerManager>::I().createTimer([this]() {
		ping();
	}, DEFAULT_PING_INTERVAL_SECONDS);
	ping();
	
		
		doUpdate();
		m_updateFinish = true;

		m_updateTimer = Singleton<TimerManager>::I().createTimer([this]() {
			onUpdate();
		}, 3600);
		

		m_fetchTimer = Singleton<TimerManager>::I().createTimer([this]() {
			doFetchTasks(false);
		}, 3600); 
		

		doFetchTasks(false);
	});
	worker.detach();
};

void ServiceApp::onStop() {
	HostFinder::setStopPolling(true);
	Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_updateTimer);
	Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_fetchTimer);
    Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_pingTimer);
	Singleton<NotiferFactory>::I().uninit();
};

void ServiceApp::onUpdate() {
	std::thread worker([this] {
		doUpdate();
	});
	worker.detach();
};

void ServiceApp::doUpdate() {

	AssistPath path;
	string command_line = path.GetCurrDir() + "\\" + UPDATER_NAME + UPDATER_COMMAND;
	Process(command_line).syncRun(120);
	return;
}



void ServiceApp::doFetchTasks(bool fromKick) {
	std::thread worker([fromKick] {
		Singleton<task_engine::TaskSchedule>::I().Fetch(fromKick);
	});
	worker.detach();
};



void  ServiceApp::onCommand(string msg) {

	if ( msg=="kick_vm" && m_updateFinish ) {
		doFetchTasks(true);
	}
	else if(msg == "shutdown"){
		doShutdown();
	}
	else if (msg == "reboot") {
		doReboot();
	}
};


void  ServiceApp::doShutdown() {
	Process("shutdown -f -s -t 0").syncRun(10);
};


void  ServiceApp::doReboot() {
	Process("shutdown -f -r -t 0").syncRun(10);
};

void ServiceApp::ping() {
    std::string virtType = OsUtils::getVirtualType();
    std::string osType = OsUtils::getOsType();
    std::string appVersion = FILE_VERSION_RESOURCE_STR;
    unsigned long startTime = OsUtils::getUptimeOfMs();
    int64_t timestamp = TimeTool::GetAccurateTime();

    std::string encodedOsVersion;
    HttpRequest::url_encode(OsVersion::GetVersion(), encodedOsVersion);

    char paramChars[512];
    sprintf(paramChars, "?virt_type=%s&os_type=%s&os_version=%s&app_version=%s&uptime=%lu&timestamp=%lld", 
        virtType.c_str(), osType.c_str(), encodedOsVersion.c_str(), appVersion.c_str(), startTime, timestamp);
    std::string params(paramChars);
    std::string url = ServiceProvide::GetPingService() + params;
    std::string response;

    int nextIntervalSeconds = DEFAULT_PING_INTERVAL_SECONDS;
    bool newTask = false;
    try {
        HttpRequest::https_request_get(url, response);
        //response = "{ \"code\": 200, \"instanceId\": \"i-abcddefg\", \"nextInterval\": 10000, \"newTask\": false }";
        Log::Info("ping request: %s, response: %s", url.c_str(), response.c_str());
        string errinfo;
        auto json = json11::Json::parse(response, errinfo);
        if (errinfo != "") {
            Log::Error("invalid json format");
        } else {
            nextIntervalSeconds = json["nextInterval"].int_value() / 1000;
            newTask = json["newTask"].bool_value();
        }
    } catch (...) {
        Log::Error("ping request url: %s got error", url.c_str());
    }
    
    task_engine::Timer* pingTimer = (task_engine::Timer*)m_pingTimer; 
    pingTimer->interval = nextIntervalSeconds < PING_INTERVAL_SECONDS_MIN ? PING_INTERVAL_SECONDS_MIN : nextIntervalSeconds;
    Singleton<TimerManager>::I().updateTime(pingTimer);
    if (newTask) {
	    doFetchTasks(false);
    }
}


void ServiceApp::runCommon() {
	onStart(0,nullptr);
	Sleep(INFINITE);
};

void ServiceApp::runService() {
	run();
};

void  ServiceApp::becomeDeamon() {
	return;
};


