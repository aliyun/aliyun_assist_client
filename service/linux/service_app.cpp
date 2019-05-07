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

#include "utils/singleton.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"
#include "utils/Log.h"
#include "utils/host_finder.h"
#include "utils/process.h"

#include "timer_manager.h"


using task_engine::TimerManager;


#define UPDATER_NAME  "aliyun_assist_update"
#define UPDATER_COMMAND " --check_update"


void ServiceApp::runService() {
    start();
};

void ServiceApp::runCommon() {
	start();
}


/*Create the Deamon Service*/
int ServiceApp::becomeDeamon()
{
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

  //signal(SIGCHLD, sig_fork);
  /* Change the current working directory */

    if ( (chdir("/root") ) < 0 ) {
       Log::Error("Failed to change working directory for AliYunAssistService: %s", strerror(errno));
       exit(EXIT_FAILURE);
    }
	
	m_updateFinish = false;
	Singleton<TimerManager>::I().start();
	
	m_notifer = Singleton<NotiferFactory>::I().createNotifer([this](const char* msg) {
		onCommand(msg);
	});

  int retryCount = 0;
  while ( HostFinder::getServerHost().empty() ) {
    int second = int(pow(2, retryCount));
    std::this_thread::sleep_for(std::chrono::seconds(second));
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
  Singleton<NotiferFactory>::I().closeNotifer(m_notifer);
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
};


void  ServiceApp::doReboot() {
	Process("/sbin/shutdown -r now").syncRun(10);
};


