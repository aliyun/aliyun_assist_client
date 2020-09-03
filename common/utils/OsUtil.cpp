#include "OsUtil.h"
#include "string.h"
#include "Log.h"

#ifdef _WIN32
#include <Windows.h>
#else 
#include <sys/sysinfo.h>
#endif

#ifdef _WIN32
unsigned long OsUtils::getUptimeOfMs() {
    // TODO , lower version??
    return GetTickCount64();
}

std::string OsUtils::getOsType() {
    return "windows";
}

#else // Linux
unsigned long OsUtils::getUptimeOfMs() {
    // TODO, how about kernel < 2.4??
    struct sysinfo s_info;
    int error = sysinfo(&s_info);
    if (error != 0) {
        return -1;
    }
    return s_info.uptime * 1000;
}

std::string OsUtils::getOsType() {
    return "linux";
}
#endif

unsigned int OsUtils::cpuid(unsigned int num, char *sig)
{
	unsigned int *sig32 = (unsigned int *)sig;
	unsigned int _eax   = num, _ebx = 0, _ecx, _edx;

#ifdef _WIN32// _WIN32
	__asm  {
		mov eax, _eax
		xchg ebx, dword ptr[_ebx]
		cpuid
		xchg ebx, dword ptr[_ebx]
		mov dword ptr[_ecx], ecx
		mov dword ptr[_edx], edx
		mov dword ptr[_eax], eax
	}
#else
	asm("cpuid"
		   : "=a"(_eax), "=b"(_ebx), "=c"(_ecx),"=d"(_edx)
	       : "a"(_eax),"b"(_ebx)
	    );
#endif 
	sig32[0] = _ebx;
	sig32[1] = _ecx;
	sig32[2] = _edx;
	sig[12]  = 0;
	return  _eax;
}

#define KVM_SIG "KVM"
#define XEN_SIG "Xen"

// copy from notifer_factory, TODO refactor notifier_factory
std::string OsUtils::getCpuSig() {

	unsigned int base = 0x40000000, leaf = base;
	unsigned int max_entries;

	char sig[128] = { 0 };
	max_entries = cpuid(leaf, sig);

	if (!strstr(sig, KVM_SIG) && !strstr(sig, XEN_SIG)) {
		for (leaf = base + 0x100; leaf <= base + max_entries; leaf += 0x100) {
			memset(sig, 0, sizeof(sig));
			cpuid(leaf, sig);

			if (strstr(sig, KVM_SIG) || strstr(sig, XEN_SIG)) {
				break;
			}
		}
	}

	Log::Info("getCpuSig sig is :%s", sig);
	return sig;
}

// TODO, return enum
std::string OsUtils::getVirtualType() {
    // use static 
    static std::string sig = OsUtils::getCpuSig();
	if (sig.find(KVM_SIG) != std::string::npos) { 
        return "kvm";
    } else if( sig.find(XEN_SIG) != std::string::npos ) {
        return "xen";
    } else {
        // TODO
        return "unknown";
    }
}
