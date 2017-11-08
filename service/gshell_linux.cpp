#include "gshell.h"
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <unistd.h>  
#include <sys/types.h>
#include <sys/ioctl.h>
#include <sys/wait.h>
#include <unistd.h>
#include <errno.h>
#include <fcntl.h>
#include <stdio.h>
#include <string.h>
#include <sys/stat.h>
#include <inttypes.h>
#include "utils/Log.h"

#define EXIT_SUCCESS    0
#define EXIT_FAILURE    1

Gshell::Gshell(KICKER kicker) {

    m_kicker = kicker;

    m_hFile = open("/dev/virtio-ports/org.qemu.guest_agent.0", 
        O_RDWR|O_NONBLOCK);

    Log::Info("/dev/virtio-ports/org.qemu.guest_agent.0:%d", m_hFile);
    if ( m_hFile > 0 ) {
        return;
    }
    Log::Error("Failed to open gshell: %s", strerror(errno));
    m_hFile = open("/dev/ttyS0", 
        O_RDWR | O_NONBLOCK | O_NOCTTY);

    Log::Info("/dev/ttyS0:%d", m_hFile);
    return;
};

Gshell::~Gshell() {
    if (m_hFile > 0) {
        close(m_hFile);
    }
}



void  Gshell::Parse(string input, string& output) {
    Log::Info("command:%s", input.c_str());
    string errinfo;
    auto json = json11::Json::parse(input, errinfo);
    if (errinfo != "") {
        return;
    };

    if (json["execute"] == "guest-sync") {
        return QmpGuestSync(json["arguments"], output);
    }

    if (json["execute"] == "guest-command") {
        return QmpGuestCommand(json["arguments"], output);
    }

    Error err;
    err.SetDesc("not suport");
    output = err.Json().dump() + "\n";
};


void  Gshell::QmpGuestSync(json11::Json  arguments, string& output) {
    json11::Json resp = json11::Json::object{ { "return", arguments["id"] } };
    output = resp.dump() + "\n";
};


void  Gshell::QmpGuestCommand(json11::Json  arguments, string& output) {

    string cmd = arguments["cmd"].string_value();
    if (arguments["cmd"] == "kick_vm" && m_kicker) {

        m_kicker();
        json11::Json   GuestCommandResult = json11::Json::object{
            { "result",8 },
            { "cmd_output", "execute kick_vm success" }
        };

        json11::Json  resp = json11::Json::object{ { "return", GuestCommandResult } };
        output = resp.dump() + "\n";

    }
    else {
        Error err;
        err.SetDesc("not suport");
        output = err.Json().dump() + "\n";
    }
};


void Gshell::reopen_fd_to_null(int fd)
{
    int nullfd;

    nullfd = open("/dev/null", O_RDWR);
    if (nullfd < 0) {
        return;
    }

    dup2(nullfd, fd);

    close(nullfd);
}

bool  Gshell::ga_wait_child(pid_t pid, int *status) {
    pid_t rpid;
    *status = 0;
    do {
        rpid = waitpid(pid, status, 0);
    } while (rpid == -1 && errno == EINTR);

    if ( rpid == -1 ) {
        return false;
    }
    return  true;
}

bool  Gshell::Poll() {

    if ( m_hFile <= 0 ) {
        return false;
    }

    char  buffer[4*1024] = {0};
    int  len = read(m_hFile, buffer, sizeof(buffer) - 1);

    if (len <= 0) {
        usleep(THREAD_SLEEP_TIME*1000);
        return true;
    }
    buffer[len] = 0;

#ifdef _DEBUG
    printf("[r]:%s\n", buffer);
#endif

    string output;
    Parse(buffer, output);
    //WriteFile(m_hFile, output.c_str(), output.length(), &len, 0);
    write(m_hFile, output.c_str(), output.length());

#ifdef _DEBUG
    printf("[w]:%s\n", output.c_str());
#endif 

    return true;
};
