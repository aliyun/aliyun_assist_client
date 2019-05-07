// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <thread>
#include <math.h>
#include "service_app.h"
#include "utils/singleton.h"
#include "./schedule_task.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/process.h"
#include "utils/Log.h"
#include "timer_manager.h"
#include "../notifer_factory.h"
#include "utils/host_finder.h"
#include "utils/process.h"


using task_engine::TimerManager;


#define UPDATER_NAME	"aliyun_assist_update.exe"
#define UPDATER_COMMAND " --check_update"


void  ServiceApp::onStart(DWORD argc, TCHAR* argv[]) {
	
	m_updateFinish = false;
	Singleton<TimerManager>::I().start();

	std::thread worker( [this] {
		
		m_notifer = Singleton<NotiferFactory>::I().createNotifer([this](const char* msg) {
			onCommand(msg);
		});

    int retryCount = 0;
    while ( HostFinder::getServerHost().empty() ) {
      int second = int(pow(2, retryCount));
      Sleep(second * 1000);
      if (retryCount < 10)
        retryCount++;
    }
		
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
	Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_updateTimer);
	Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_updateTimer);
	Singleton<NotiferFactory>::I().closeNotifer(m_notifer);
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


