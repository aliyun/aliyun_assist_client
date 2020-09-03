#include "ProcessSingleton.h"
#include <fstream>
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"

namespace ProcessSingleton {

Lock::Lock(const std::string &lockName)
    : m_lockName(lockName) {
#if defined(_WIN32)
    this->m_windowsLockHandle = nullptr;
#else
    this->m_unixLockPath = std::string("/run/") + lockName + ".lock";
    this->m_unixLockFd = -1;
#endif
}

Lock::~Lock() {
    this->unlock();
}

bool Lock::tryLock() {
#if defined(_WIN32)
#if defined(UNICODE)
    // CreateMutexW needs LPCWSTR for 3rd parameter, and prepend global namespace
    // prefix for creating mutex object across sessions on servers
    std::wstring globalLockName(L"Global\\");
    // Simply copy each char as wchar which only applies to single-byte string.
    // See https://stackoverflow.com/a/8969776
    globalLockName.append(this->m_lockName.begin(), this->m_lockName.end());
#else
    // Prepend global namespace prefix for creating mutex object across sessions on servers
    std::string globalLockName("Global\\");
    globalLockName.append(this->m_lockName);
#endif
    this->m_windowsLockHandle = ::CreateMutex(nullptr, TRUE, globalLockName.c_str());
    if (this->m_windowsLockHandle != nullptr && ::GetLastError() != ERROR_ALREADY_EXISTS) {
        return true;
    }
    return false;
#else
    this->m_unixLockFd = ::open(this->m_unixLockPath.c_str(), O_CREAT | O_RDWR | O_CLOEXEC, 0644);
	if (this->m_unixLockFd == -1) {
		return false;
	}
    // Some case of co-existence of aliyun-service processes happens on CentOS 5,
    // which uses kernel 2.6.18 that does not support O_CLOEXEC. Therefore, we
    // MUST check FD_CLOEXEC manually.
    if ((::fcntl(this->m_unixLockFd, F_GETFD) & FD_CLOEXEC) == 0) {
        if (::fcntl(this->m_unixLockFd, F_SETFD, FD_CLOEXEC) != 0) {
            return false;
        }
    }

    if (::flock(this->m_unixLockFd, LOCK_EX | LOCK_NB) == 0) {
        return true;
    }

    ::close(this->m_unixLockFd);
    this->m_unixLockFd = -1;
    return false;
#endif
}

void Lock::unlock() {
#if defined(_WIN32)
    if (this->m_windowsLockHandle != nullptr) {
        ::ReleaseMutex(this->m_windowsLockHandle);
        ::CloseHandle(this->m_windowsLockHandle);
    }
#else
    if (this->m_unixLockFd >= 0) {
        // Releasing lock via flock() does not guanrante to succeed, but I have
        // not found graceful ways to handle such error.
        ::flock(this->m_unixLockFd, LOCK_UN | LOCK_NB);
        ::close(this->m_unixLockFd);
        this->m_unixLockFd = -1;
        FileUtils::removeFile(this->m_unixLockPath.c_str());
    }
#endif
}

PidHolder::PidHolder(const std::string &holderName)
    : m_holderName(holderName), m_holderPath(), m_pidSaved(false) {
    this->m_holderPath = PidHolder::generateHolderPath(holderName);
}

PidHolder::~PidHolder() {
    if (FileUtils::fileExists(this->m_holderPath.c_str()) && this->m_pidSaved) {
        this->unHold();
    }
}

bool PidHolder::tryHold() {
#if defined(_WIN32)
    DWORD pid = ::GetCurrentProcessId();
#else
    pid_t pid = ::getpid();
#endif

    std::ofstream pidFile(this->m_holderPath);
    pidFile << pid;
    pidFile.close();

    this->m_pidSaved = true;
    return true;
}

void PidHolder::unHold() {
    this->m_pidSaved = false;
    // Pidfile should not be deleted by other aliyun-service processes since
    // process lock must be acquired at first.
    FileUtils::removeFile(this->m_holderPath.c_str());
}

const std::string PidHolder::getHolderPath() const {
    return this->m_holderPath;
}

std::string PidHolder::generateHolderPath(const std::string &holderName) {
#if defined(_WIN32)
    AssistPath assistPath("");
    std::string path = assistPath.GetCrossVersionWorkPath() + '\\' + holderName + ".pid";
    return path;
#else
    return std::string("/run/") + holderName + ".pid";
#endif
}

std::string PidHolder::getRunningPid(const std::string &holderName) {
    std::string holderPath = PidHolder::generateHolderPath(holderName);

    if (!FileUtils::fileExists(holderPath.c_str())) {
        return "Unknown locking process";
    }

    std::ifstream pidFile(holderPath);
    std::string pidFileContent;
    pidFile >> pidFileContent;
    return pidFileContent;
}

}
