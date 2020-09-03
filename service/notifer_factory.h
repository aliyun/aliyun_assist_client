#pragma once

#include <string>
#include <functional>
#include "utils/singleton.h"

#ifdef _WIN32
#include "windows/kvm_notifer.h"
#include "windows/xen_notifer.h"
#else
#include "linux/kvm_notifer.h"
#include "linux/xen_notifer.h"
#endif
#include "wskt_notifer.h"
#include <memory>

using std::string;
using std::function;
class  NotiferFactory {
	friend Singleton<NotiferFactory>;
public:
	void createNotifer();
	void init(function<void(const char*)> callback);
	void uninit();
	void enableWebsocket();
	void disableWebsocket();
private:
	unsigned int cpuid(unsigned int num, char *sig);
	//NotiferFactory() {};
	string	     getCpuSig();
	void  onCheck();
	static void* doCheck(void* args);
	function<void(const char*)> notify_callback_;
	void*   check_timer_;
	std::unique_ptr<KvmNotifer> kvm_notifier_;
	std::unique_ptr<XenNotifer> xen_notifier_;
	std::unique_ptr<WsktNotifer> wskt_notifier_;
};