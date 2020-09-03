#pragma once
#include "service_base.h"
#include "utils/singleton.h"
class ServiceApp : public ServiceBase {
	friend Singleton<ServiceApp>;
public:
	ServiceApp() :ServiceBase(_T("aliyun_assist_service")) {};

	void  becomeDeamon();
	void  runService();
	void  runCommon();
private:
	void	onStart(DWORD argc, TCHAR* argv[]);
	void	onStop();
	void    onCommand(std::string msg);
	void    onUpdate();
	void    doFetchTasks(bool fromKick);
	void    doUpdate();
	void    doShutdown();
	void    doReboot();
    void    ping();

private:
	void*   m_updateTimer;
	void*   m_fetchTimer;
	bool    m_updateFinish;
    void*   m_pingTimer;
};
