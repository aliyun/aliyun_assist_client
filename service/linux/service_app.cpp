// Copyright (c) 2017-2018 Alibaba Group Holding Limited

#include "service_app.h"
#include <fcntl.h>
#include <unistd.h>
#include <stdio.h>
#include <math.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <functional>
#include "../notifer_factory.h"
#include "./schedule_task.h"
#include "../VersionInfo.h"
#include "json11/json11.h"

#include "utils/singleton.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/Log.h"
#include "utils/host_finder.h"
#include "utils/process.h"
#include "utils/OsUtil.h"
#include "utils/OsVersion.h"
#include "utils/service_provide.h"
#include "utils/TimeTool.h"
#include "utils/http_request.h"
#include "utils/ProcessSingleton.h"
#include "plugin/debug_script.h"
#include "timer_manager.h"

using task_engine::TimerManager;


#define UPDATER_NAME  "aliyun_assist_update"
#define UPDATER_COMMAND " --check_update"
#define DEFAULT_PING_INTERVAL_SECONDS 1800
extern function<void(char*)>    wskt_callback;
 
#define PING_INTERVAL_SECONDS_MIN 30

#define PROCESS_SINGLETON_IDENTIFIER "aliyun-service"

void ServiceApp::runService() {
    start();
};

void ServiceApp::runCommon() {
	start();
}


/*Create the Deamon Service*/
int ServiceApp::becomeDeamon()
{
  ProcessSingleton::Lock runningAgentDetector(PROCESS_SINGLETON_IDENTIFIER);
  if (!runningAgentDetector.tryLock()) {
    Log::Error("Agent has been running with pid %s",
      ProcessSingleton::PidHolder::getRunningPid(PROCESS_SINGLETON_IDENTIFIER).c_str());
    // Immediate exiting looks not so graceful.
    exit(EXIT_FAILURE);
    return EXIT_FAILURE;
  }
  runningAgentDetector.unlock();

	pid_t pid, sid;
	int i = 0;
	struct sigaction sigActionMask;

	/* Fork off the parent process and exit the parent process*/
	pid = fork();
	if (pid < 0) {
		Log::Info("pid < 0 quit");
		exit(EXIT_FAILURE);
	}
	else if (pid > 0) {
		Log::Info("pid > 0 quit");
		exit(EXIT_SUCCESS);
	}

	Log::Info("deamon running");
	/*Set the default file access mask*/
	umask(S_IRWXG | S_IRWXO);

	/* Create a new SID for the child process */
	sid = setsid();
	if (sid < 0) {
		Log::Error("Failed to create Session for AliYunAssistService child process: %s", strerror(errno));
		exit(EXIT_FAILURE);
	}


	/* Close out the standard file descriptors */
	reopen_fd_to_null(STDIN_FILENO);
	reopen_fd_to_null(STDOUT_FILENO);
	reopen_fd_to_null(STDERR_FILENO);

	return 0;
}


void  ServiceApp::start() {
  ProcessSingleton::Lock agentLock(PROCESS_SINGLETON_IDENTIFIER);
  if (!agentLock.tryLock()) {
    Log::Error("Agent has been running with pid: %s",
      ProcessSingleton::PidHolder::getRunningPid(PROCESS_SINGLETON_IDENTIFIER).c_str());
    return;
  }
  ProcessSingleton::PidHolder agentPidHolder(PROCESS_SINGLETON_IDENTIFIER);
  if (!agentPidHolder.tryHold()) {
    Log::Error("Failed to save pid of current service to %s", agentPidHolder.getHolderPath().c_str());
    return;
  }
  // Cleaning of process lock and pidfile during normal exiting is guaranteed by RAII.
  // Releasing process lock during abnormal exiting is promised by system, and
  // pidfile may be left on disk.

  //signal(SIGCHLD, sig_fork);
  /* Change the current working directory */

    if ( (chdir("/root") ) < 0 ) {
       Log::Error("Failed to change working directory to /root for AliYunAssistService: %s", strerror(errno));
       exit(EXIT_FAILURE);
    }
	
	m_updateFinish = false;
	Singleton<TimerManager>::I().start();

  wskt_callback = [this](const char* msg) {
		onCommand(msg);
  };
	
	Singleton<NotiferFactory>::I().init([this](const char* msg) {
		onCommand(msg);
	});

  int retryCount = 0;
  while ( HostFinder::getServerHost().empty() ) {
    int second = int(pow(2, retryCount));
    std::this_thread::sleep_for(std::chrono::seconds(second));
    retryCount++;
    if (retryCount > 3)
      break;
  }
  
  if(HostFinder::getServerHost().empty()) {
    Log::Error("network internal error");
    // ubuntu18 dns available slowly, try to restart to fix it.
    Process("systemctl restart aliyun.service").syncRun(10);
  }

    m_pingTimer = Singleton<TimerManager>::I().createTimer([this]() {
		ping();
	}, DEFAULT_PING_INTERVAL_SECONDS);
	ping();
	
	doUpdate();
	m_updateFinish = true;

	m_updateTimer = Singleton<TimerManager>::I().createTimer([this]() {
		onUpdate();
	}, 1800);

	m_fetchTimer = Singleton<TimerManager>::I().createTimer([this]() {
		doFetchTasks(false);
	}, 3600);

	doFetchTasks(false);
	
	while ( true ) {
		std::this_thread::sleep_for(std::chrono::seconds(3));
	}	
}


void ServiceApp::reopen_fd_to_null(int fd) {
  int nullfd;

  nullfd = open("/dev/null", O_RDWR);
  if (nullfd < 0) {
    return;
  }

  dup2(nullfd, fd);

  close(nullfd);
}

void  ServiceApp::onCommand(string msg) {
	if (msg == "kick_vm") {
		doFetchTasks(true);
	}
	else if (msg == "shutdown") {
		doShutdown();
	}
	else if (msg == "reboot") {
		doReboot();
	}
}


void ServiceApp::doFetchTasks(bool fromKick) {
  
	auto worker =[](void* args)->void * {
		Singleton<task_engine::TaskSchedule>::I().Fetch((bool)args);
		return NULL;
	};
	
	pthread_t  ptid;
	pthread_create(&ptid, NULL, worker, (void*)fromKick);
	pthread_detach(ptid);
}

void ServiceApp::onUpdate() {
	
	auto worker = [](void* args)->void * {
		((ServiceApp*)args)->doUpdate();
		return NULL ;
	};
	
	pthread_t  ptid;
	pthread_create(&ptid, NULL, worker, NULL);
	pthread_detach(ptid);
}

void ServiceApp::onStop() {
  Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_updateTimer);
  Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_fetchTimer);
  Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_updateTimeoutTimer);
  Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)m_pingTimer);
  Singleton<NotiferFactory>::I().uninit();
}

void ServiceApp::doUpdate() {
  AssistPath path_service("");
  std::string update_path = path_service.GetCurrDir() + "";
  update_path += FileUtils::separator();
  update_path += UPDATER_NAME;
  std::string update_command = update_path + " " + UPDATER_COMMAND;
  
  Process::RunResult result =  Process(update_command).syncRun(120);
  if ( result != Process::sucess ) {
	  Log::Error("failed do update: %d", result);
  }
}



void  ServiceApp::doShutdown() {
	Process("/sbin/shutdown -h now").syncRun(10);
}


void  ServiceApp::doReboot() {
	Process("/sbin/shutdown -r now").syncRun(10);
}

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
        bool ret = HttpRequest::https_request_get(url, response);

        for (int i = 0; i < 3 && !ret; i++) {
          int second = int(pow(2, i));
          std::this_thread::sleep_for(std::chrono::seconds(second));
          ret = HttpRequest::https_request_get(url, response);
        }
        if (!ret) {
            Log::Error("assist network is wrong");
            task_engine::DebugTask task;
            task.RunSystemNetCheck();
        }
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
