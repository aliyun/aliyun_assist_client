// Copyright (c) 2017-2018 Alibaba Group Holding Limited


#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/wait.h>
#include <string>
#include <thread>
#include "kvm_notifer.h"
#include "utils/Log.h"

struct Error {
	Error() {
		m_class = "GenericError";
		m_decs = "";
	}

	void  setClass(string Class) {
		m_class = Class;
	}

	void  setDesc(string Desc) {
		m_decs = Desc;
	}

	json11::Json  toJson() {
		return json11::Json::object{ {
				"error",
				json11::Json::object{
					{ "class", m_class },
					{ "desc",  m_decs }
				}
			} };
	}
private:
	string  m_class;
	string  m_decs;
};


KvmNotifer::KvmNotifer() {
	m_hFile  = -1;
	m_worker = nullptr;
	m_stop = true;
};


bool KvmNotifer::init(function<void(const char*)> callback) {
  int retry = 3;
  while (retry > 0) {
    m_hFile = open("/dev/virtio-ports/org.qemu.guest_agent.0",
      O_RDWR | O_NONBLOCK | O_CLOEXEC);
    Log::Info("/dev/virtio-ports/org.qemu.guest_agent.0:%d", m_hFile);
    if (m_hFile == -1) {
      Log::Error("Failed to open gshell: %s", strerror(errno));
      retry--;
      sleep(1);
    } else {
      break;
    }
  }
  
  if (m_hFile == -1) {
    return false;
  }

  fcntl(m_hFile, F_SETFD, FD_CLOEXEC);

	m_callback = callback;
	m_stop     = false;
	m_worker   = new thread([this]() {
		poll();
	});

	return true;
}

void KvmNotifer::unit() {
  m_stop = true;
  if (m_worker) {
	  m_worker->join();
  }

  if (m_hFile > 0) {
    close(m_hFile);
    m_hFile = -1;
  }
}

bool  KvmNotifer::poll() {
  while (!m_stop) {
    char  buffer[1024] = { 0 };
    int  len = read(m_hFile, buffer, sizeof(buffer) - 1);

    if ( len <= 0 ) {
	    std::this_thread::sleep_for(std::chrono::milliseconds(200)); 
	    continue;
    }
    buffer[len] = 0;

#ifdef _DEBUG
    printf("[r]:%s\n", buffer);
#endif

    string output;
    parse(buffer, output);
    Log::Info("[w]:%s\n", output.c_str());
    write(m_hFile, output.c_str(), output.length());
#ifdef _DEBUG
    printf("[w]:%s\n", output.c_str());
#endif 
  }
  return true;
}

void  KvmNotifer::parse(string input, string& output) {
  Log::Info("command:%s", input.c_str());
  string errinfo;
  auto json = json11::Json::parse(input, errinfo);
  if ( errinfo != "" ) {
    return;
  }

  if (json["execute"] == "guest-sync") {
      return onGuestSync(json["arguments"], output);
  }

  if (json["execute"] == "guest-command") {
      return onGuestCommand(json["arguments"], output);
  }

#if !defined(GSHELL_NOT_SUPPORT_SHUTDOWN)
  if (json["execute"] == "guest-shutdown") {
    return onGuestShutdown(json["arguments"], output);
  }
#endif

  Error err;
  err.setDesc("not suport");
  output = err.toJson().dump() + "\n";
}

// gshell check ready
/*{ 'command': 'guest-sync',
'data' : { 'id': 'int' },
'returns' : 'int' }*/
void KvmNotifer::onGuestSync( json11::Json  arguments, string& output ) {
    json11::Json resp = json11::Json::object{ { "return", arguments["id"] } };
    output = resp.dump() + "\n";
}

/*
{ 'command': 'guest-command',
'data': { 'cmd': 'str', 'timeout': 'int' },
'returns': 'GuestCommandResult' }

{ 'type': 'GuestCommandResult',
'data': { 'result': 'int', 'cmd_output': 'str' } }
*/

void  KvmNotifer::onGuestCommand(json11::Json  arguments, string& output) {
  string cmd = arguments["cmd"].string_value();
  if (arguments["cmd"] == "kick_vm" ) {
	Log::Info("receive task notify");
	m_callback("kick_vm");
    json11::Json  GuestCommandResult = json11::Json::object{
        { "result",8 },
        { "cmd_output", "execute kick_vm success" }
    };
    json11::Json  resp = json11::Json::object{ { "return",
        GuestCommandResult } };
    output = resp.dump() + "\n";
  } else {
    Error err;
    err.setDesc("not suport");
    output = err.toJson().dump() + "\n";
  }
}

void  KvmNotifer::onGuestShutdown(json11::Json arguments, string& output) {
  const char *shutdown_flag;
  Error err;
  pid_t pid;
  int status;

  if (arguments["mode"].is_null()) {
    err.setDesc("powerdown|reboot");
    output = err.toJson().dump() + "\n";
    return;
  }
  bool  bRebootAfterShutdown;

  if (arguments["mode"].string_value() == "powerdown") {
	  bRebootAfterShutdown = false;
  }
  else if (arguments["mode"].string_value() == "reboot") {
	  bRebootAfterShutdown = true;
  }
  else {
	  err.setDesc("powerdown|reboot");
	  output = err.toJson().dump() + "\n";
	  return;
  }

  if ( bRebootAfterShutdown) {
	  m_callback("reboot");
  }
  else {
	  m_callback("shutdown");
  }

  json11::Json   GuestCommandResult = json11::Json::object{
    { "result",8 },
    { "cmd_output", "execute command success" }
  };
  json11::Json resp = json11::Json::object{ { "return", GuestCommandResult } };
  output = resp.dump() + "\n";
}

