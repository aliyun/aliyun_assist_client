#include "task_notifer.h"
#include "notifer_factory.h"
#include "string.h"
#include "utils/Log.h"

#include <memory>
#include "timer_manager.h"
#include "utils/http_request.h"
#include "utils/service_provide.h"
#include "utils/host_finder.h"

using task_engine::TimerManager;

#define KVM_SIG "KVM"
#define XEN_SIG "Xen"


void NotiferFactory::init(function<void(const char*)> callback) {
	notify_callback_ = callback;
  //enableWebsocket();
	createNotifer();

	doCheck(this);
}

void NotiferFactory::uninit() {
	Singleton<TimerManager>::I().deleteTimer((task_engine::Timer*)check_timer_);
	if (wskt_notifier_.get()) {
		wskt_notifier_.get()->unit();
	}
	if (kvm_notifier_.get()) {
		kvm_notifier_.get()->unit();
	}
	if (xen_notifier_.get()) {
		xen_notifier_.get()->unit();
	}
}


void NotiferFactory::onCheck() {
#ifdef _WIN32
	std::thread(doCheck, (void*) this).detach(); 
#else
	pthread_t tid;
	pthread_create(&tid, nullptr, doCheck, (void*) this);
	pthread_detach(tid);
#endif
}


static std::string ParseGshellStatusInfo(std::string response) {
	try {
		string errinfo;
		auto json = json11::Json::parse(response, errinfo);
		if (errinfo != "") {
			Log::Error("invalid gshell json format");
			return "false";
		}

		return json["gshellSupport"].string_value();
		;

	}
	catch (...) {
		Log::Error("gshell status json is invalid");
		return "false";
	}
}

void NotiferFactory::enableWebsocket() {
	if (wskt_notifier_.get() == nullptr) {
		wskt_notifier_ =  std::unique_ptr<WsktNotifer>(new WsktNotifer());
		wskt_notifier_.get()->init(nullptr);
		Log::Info("enable the web socket");
	}
}

void NotiferFactory::disableWebsocket() {
	if (wskt_notifier_.get()) {
		wskt_notifier_.get()->unit();
		Log::Info("disable the web socket");
		wskt_notifier_.reset();
	}
}

void* NotiferFactory::doCheck(void* args) {
	NotiferFactory* pthis = (NotiferFactory*) args;
	if (HostFinder::getServerHost().empty()) {
		return nullptr;
	}
	bool gshell_status = false;
	std::string response;
	std::string url = ServiceProvide::GetGshellCheckService();

	bool ret = HttpRequest::https_request_get(url, response);
	if (!ret) {
		Log::Error("check gshell status failed %s", response.c_str());
	}
	else {
		if (ParseGshellStatusInfo(response) == "true") {
			gshell_status = true;
		}
	}

  Log::Info("check gshell status %d", gshell_status);

	  // enable the web socket
	if (!gshell_status) {
		pthis->enableWebsocket();
	}
	else {
		pthis->disableWebsocket();
	}
	return nullptr;
}


void NotiferFactory::createNotifer() {
  //asm("int $3"::);
	string sig = getCpuSig();
	if (sig.find(KVM_SIG) != string::npos) {
		kvm_notifier_ =  std::unique_ptr<KvmNotifer>(new KvmNotifer());
		kvm_notifier_.get()->init(notify_callback_);
	}
	else if (sig.find(XEN_SIG) != string::npos) {
		xen_notifier_ = std::unique_ptr<XenNotifer>(new XenNotifer());
		xen_notifier_.get()->init(notify_callback_);
	}
	else {
		wskt_notifier_ = std::unique_ptr<WsktNotifer>(new WsktNotifer());
		wskt_notifier_.get()->init(notify_callback_);
	}
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
	__asm {
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
	    : "=a"(_eax),
		"=b"(_ebx),
		"=c"(_ecx),
		"=d"(_edx)
	       : "a"(_eax),
		"b"(_ebx));
#endif 
	sig32[0] = _ebx;
	sig32[1] = _ecx;
	sig32[2] = _edx;
	sig[12]  = 0;
	return _eax;
}