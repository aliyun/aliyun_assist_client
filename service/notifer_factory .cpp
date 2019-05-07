#include "task_notifer.h"
#include "notifer_factory.h"
#include "string.h"
#include "utils/Log.h"

#ifdef _WIN32
#include "windows/kvm_notifer.h"
#include "windows/xen_notifer.h"
#else
#include "linux/kvm_notifer.h"
#include "linux/xen_notifer.h"
#endif
#include "wskt_notifer.h"


#define KVM_SIG "KVM"
#define XEN_SIG "Xen"

void* NotiferFactory::createNotifer(function<void(const char*)> callback)  {
	//asm("int $3"::);
	string sig = getCpuSig();
	if ( sig.find(KVM_SIG) != string::npos ) {
		KvmNotifer* notifier =  new KvmNotifer();
		if ( notifier->init(callback) ){
			return notifier;
		}
	}
	else if( sig.find(XEN_SIG) != string::npos ) {
		XenNotifer* notifier = new XenNotifer();
		if (notifier->init(callback)) {
			return notifier;
		}
	}
	else{
		WsktNotifer* notifier = new WsktNotifer();
		if ( notifier->init(callback) ){
			return notifier;
		}
	}
	return nullptr;
}

void NotiferFactory::closeNotifer(void *notifier) {
	((TaskNotifer*)notifier)->unit();
}


string NotiferFactory::getCpuSig() {

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



unsigned int NotiferFactory::cpuid(unsigned int num, char *sig)
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