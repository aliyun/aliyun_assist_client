#ifndef COMMON_UTILS_PROCESS_SINGLETON_H_
#define COMMON_UTILS_PROCESS_SINGLETON_H_
#include <string>

#if defined(_WIN32)
#define WIN32_LEAN_AND_MEAN
#include "windows.h"
#else
#include <sys/types.h>
#include <sys/file.h>
#include <unistd.h>
#endif

namespace ProcessSingleton {

class Lock {
public:
    Lock(const std::string &lockName);
    ~Lock();
    bool tryLock();
    void unlock();

private:
    std::string m_lockName;
#if defined(_WIN32)
    HANDLE m_windowsLockHandle;
#else
    std::string m_unixLockPath;
    int m_unixLockFd;
#endif

// Delete other constructors for non copy-assignable
public:
    Lock(const Lock &) = delete;
    Lock(Lock &&) = delete;
    Lock & operator=(const Lock &) = delete;
    Lock & operator=(Lock &&) = delete;
};

class PidHolder {
public:
    PidHolder(const std::string &holderName);
    ~PidHolder();
    bool tryHold();
    void unHold();
    const std::string getHolderPath() const;

public:
    static std::string generateHolderPath(const std::string &holderName);
    static std::string getRunningPid(const std::string &holderName);

private:
    std::string m_holderName;
    std::string m_holderPath;
    bool m_pidSaved;

// Delete other constructors for non copy-assignable
public:
    PidHolder(const PidHolder &) = delete;
    PidHolder(PidHolder &&) = delete;
    PidHolder & operator=(const PidHolder &) = delete;
    PidHolder & operator=(PidHolder &&) = delete;
};

}

#endif  // COMMON_UTILS_PROCESS_SINGLETON_H_
