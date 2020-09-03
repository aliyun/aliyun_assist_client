#pragma once
#include <string>

class OsUtils {
public:
    /**
     * uptime:milliseconds 
     */
    static unsigned long getUptimeOfMs();

    /**
     * linux / windows, TODO enum
     */
    static std::string getOsType();

    /**
     * kvm / xen / ??, TODO enum
     */
    static std::string getVirtualType();

private:
    static unsigned int cpuid(unsigned int num, char *sig);
    static std::string getCpuSig();
};
