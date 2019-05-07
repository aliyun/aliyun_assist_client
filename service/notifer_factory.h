#pragma once

#include <string>
#include <functional>
#include "utils/singleton.h"

using std::string;
using std::function;
class  NotiferFactory {
	friend Singleton<NotiferFactory>;
public:
	void* createNotifer(function<void(const char*)> callback) ;
	void  closeNotifer(void *notifer);
private:
	unsigned int cpuid(unsigned int num, char *sig);
	//NotiferFactory() {};
	string	     getCpuSig();
};