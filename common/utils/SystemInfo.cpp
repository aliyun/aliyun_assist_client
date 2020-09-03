#include "SystemInfo.h"

#ifdef _WIN32
//#define WIN32_LEAN_AND_MEAN
#include <winsock2.h>
#include <stdio.h>
#include <iostream>
#include <cstring>
#include <ws2tcpip.h>
//#pragma comment(lib, "ws2_32.lib")
//#include <ws2tcpip.h>
//#include <stdio.h>
/*#define WIN32_LEAN_AND_MEAN
#include <iostream>
#include <vector>
#include <WinSock2.h>
#include <Iphlpapi.h>*/
#else
#include <stdio.h>
#include <sys/types.h>
#include <ifaddrs.h>
#include <netinet/in.h>
#include <string.h>
#include <arpa/inet.h>
#endif

#include "Log.h"
#include "MutexLocker.h"

std::string SystemInfo::GetAllIPs() {
  static std::string all_ips;
  static std::mutex  mutex;
  MutexLocker(&mutex) {
    if (!all_ips.empty()) {
      return all_ips;
    }
    
    int ip_count = 0;

#ifdef _WIN32
    if (InitWSA()) {
      PHOSTENT hostinfo;
      char hostname[255] = { 0 };
      gethostname(hostname, sizeof(hostname));
      if ((hostinfo = gethostbyname(hostname)) == NULL) {
        errno = GetLastError();
        Log::Error("gethostbyname Error:%d", errno);
        ReleaseWSA();
      }

      LPCSTR ip;
      while (*(hostinfo->h_addr_list) != NULL) {
        if(ip_count >= 3) {
          break;
        }
        ip = inet_ntoa(*(struct in_addr *) *hostinfo->h_addr_list);
        Log::Info("ipv4 address: %s", ip);
        if (!all_ips.empty())
          all_ips += ";";
        all_ips += ip;
        hostinfo->h_addr_list++;
        ip_count++;
      }
      ReleaseWSA();
    }
#else
    struct ifaddrs * ifAddrStruct = NULL, *ifAddrHead = NULL;
    void * tmpAddrPtr = NULL;

    if (getifaddrs(&ifAddrHead) == -1) {
        Log::Error("getifaddrs return -1");
        return all_ips;
    }

    for (ifAddrStruct = ifAddrHead; ifAddrStruct != NULL; ifAddrStruct = ifAddrStruct->ifa_next) {
      std::string name = ifAddrStruct->ifa_name;
      if(ip_count >= 3) {
        break;
      }
      if (name == "lo" || ifAddrStruct->ifa_addr == NULL) {
        continue;
      }
      if (ifAddrStruct->ifa_addr->sa_family == AF_INET) { // check it is IP4
                                                          // is a valid IP4 Address
        tmpAddrPtr = &((struct sockaddr_in *)ifAddrStruct->ifa_addr)->sin_addr;
        char addressBuffer[INET_ADDRSTRLEN];
        inet_ntop(AF_INET, tmpAddrPtr, addressBuffer, INET_ADDRSTRLEN);
        Log::Info("%s IP Address %s", ifAddrStruct->ifa_name, addressBuffer);
        if (!all_ips.empty())
          all_ips += ";";
        all_ips += addressBuffer;
        ip_count++;
      } else if (ifAddrStruct->ifa_addr->sa_family == AF_INET6) { // check it is IP6
                                                                // is a valid IP6 Address
        tmpAddrPtr = &((struct sockaddr_in *)ifAddrStruct->ifa_addr)->sin_addr;
        char addressBuffer[INET6_ADDRSTRLEN];
        inet_ntop(AF_INET6, tmpAddrPtr, addressBuffer, INET6_ADDRSTRLEN);
        Log::Info("%s IP Address %s", ifAddrStruct->ifa_name, addressBuffer);
        if (!all_ips.empty())
          all_ips += ";";
        all_ips += addressBuffer;
        ip_count++;
      }
    }
    freeifaddrs(ifAddrHead);
#endif // _WIN32
    return all_ips;
  }
}

#ifdef _WIN32
bool SystemInfo::InitWSA() {
  WORD wVersionRequested;
  WSADATA wsaData;
  int err;
  wVersionRequested = MAKEWORD(1, 1);
  err = WSAStartup(wVersionRequested, &wsaData);//initiate the ws2_32.dll and match the version
  if (err != 0) {
    Log::Error("WSAStartup error: %d", err);
    return false;
  }

  if (LOBYTE(wsaData.wVersion) != 1 ||   //if the version is not matched ,then quit and terminate the ws3_32.dll 
    HIBYTE(wsaData.wVersion) != 1) {
    Log::Error("wsa version not matched");
    WSACleanup();
    return false;
  }

  return true;
}

void SystemInfo::ReleaseWSA() {
  WSACleanup();
}

unsigned long SystemInfo::GetWindowsDefaultLang() {
	static LCID s_lcid = 0;
	if (0 == s_lcid) {
		s_lcid = GetSystemDefaultLCID();
	}
	return s_lcid;
}
#endif // _WIN32
