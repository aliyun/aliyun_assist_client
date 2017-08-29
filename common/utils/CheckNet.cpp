/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-14
Type: .cpp
Description: Provide functions to resolve server address and check net
**************************************************************************/
#ifdef _WIN32
#include <Windows.h>
#include <Iphlpapi.h>
#pragma comment(lib,"Iphlpapi.lib")
#else
#include <net/route.h>
#endif // _WIN32

#include "CheckNet.h"

#include <algorithm>

#include "utils/http_request.h"
#include "json11/json11.h"
#include "TimeTool.h"
#include "SubProcess.h"
#include "Log.h"

using namespace json11;

string HostChooser::m_HostSelect;

/*
*Summary: to get server address
*Parameters: (string&) ip
(this ip must is a server ip of available, and local pc can connect, the last back up this ip)
*Return: (bool) status
*/
bool HostChooser::Init(string path)  {

  // if ( !WaitNetworkReady(60) )  return false;

  string CfgFile = path + "/host.conf";
  string select  = path + "/host.select";

  string host;
  bool   status  = FileRead(select, host);
  if ( status && host.length() ) {
    if (HttpRequest::DetectHost(host)) {
      m_HostSelect = host;
      return true;
    }
  }

  if ( FindServer(CfgFile, host) ) {
    FileWrite(select, host);
    m_HostSelect = host;
    return true;
  }
  return false;
}

/*
*Summary: to find available server address
*Parameters: (string) file, (string&) ip
*Return: (bool) status
*/
bool HostChooser::FindServer(string file, string &ip)  {
  string content;
  if (!FileRead(file, content)) {
    Log::Error("Read file Error!");
    return false;
  }
  std::replace(content.begin(), content.end(), '\r', ' ');
  std::replace(content.begin(), content.end(), '\n', ' ');
  if (!ChooseAddress(content, ip)) {
    Log::Info("Have no available server address!");
    return false;
  }

  return true;
}

/*
*Summary: to choose server address
*Parameters: (string) content, (string&) ip
*Return: (bool) status
*/
bool HostChooser::ChooseAddress(string content, string &host)  {
  string  err;
  auto    json = Json::parse(content, err);

  for (auto &item : json["hosts"].array_items()) {
    host = item["host"].string_value();
    if (HttpRequest::DetectHost(host)) {
      Log::Info("Choose address:" +host);
      return true;
    }
  }
  return false;
}


bool HostChooser::FileRead(string file, string &content) {

  Log::Info( "Read file :" + file );
  FILE *fp = fopen(file.c_str(), "r");
  if (!fp) {
    Log::Error("File open error!");
    return false;
  }

  char *pBuf = nullptr;
  fseek(fp, 0, SEEK_END);
  long len = ftell(fp); //获取文件长度
  if(len < 0) {
    fclose(fp);
    return false;
  }
  pBuf = new char[len + 1];
  memset(pBuf, 0, sizeof(char) * (len+1));

  fseek(fp, 0, SEEK_SET);
  int count = fread(pBuf, sizeof(char), len, fp);

  pBuf[len] = 0;
  fclose(fp);

  content = pBuf;
  delete[] pBuf;
  return true;
}

bool HostChooser::FileWrite(string file, string content) {

  Log::Info("Write file :" + file);
  FILE *fp = fopen(file.c_str(), "w");;
  if (!fp) {
	Log::Error("File open error!");
	return false;
  }

  fwrite(content.c_str(),1,content.length(),fp);
  fflush(fp);
  fclose(fp);
  return true;
}


bool HostChooser::WaitNetworkReady(int timeout) {
  string gateway;
  if ( !GetAdaptInfo(gateway) ) {
    Log::Error("Get adapt info is Error!");
    return false;
  }

  int    spantime = 0;
  long   code;
  string out;
  string cmd = "ping " + gateway;

  Log::Info(cmd);
  while ( true ) {
    time_t start = Time::GetCurreTime();
    SubProcess(cmd).Execute(out, code);
    if ( code == 0 ) {
      Log::Info( gateway +" is OK");
      return true;
    }
    time_t end = Time::GetCurreTime();
    spantime += Time::GetDiffTime(start, end);
    if (spantime >= timeout) return false;
  }

  Log::Error( gateway +"is error!");
  return false;
}

bool HostChooser::GetAdaptInfo(string &GatewayIP) {

#ifdef _WIN32
  PIP_ADAPTER_INFO pIpAdapterInfo = new IP_ADAPTER_INFO();
  unsigned long stSize = sizeof(IP_ADAPTER_INFO);
  int ret = GetAdaptersInfo(pIpAdapterInfo, &stSize);
  int netCardNum = 0;//记录网卡数量
  int IPnumPerNetCard = 0;//记录每张网卡上的IP地址数量
  if ( ret == ERROR_BUFFER_OVERFLOW ) { //内存不足释放重新申请
    delete pIpAdapterInfo;
    pIpAdapterInfo = (PIP_ADAPTER_INFO)new BYTE[stSize];
    ret = GetAdaptersInfo(pIpAdapterInfo, &stSize);
  }

  if ( ret == ERROR_SUCCESS ) {
    //输出网卡信息
    while (pIpAdapterInfo) {
      //cout << "网卡名称：" << pIpAdapterInfo->AdapterName << endl;
      //cout << "网卡描述：" << pIpAdapterInfo->Description << endl;

      char *AdapterDescSrc = pIpAdapterInfo->Description;

      if (strstr(AdapterDescSrc, "Dell") || strstr(AdapterDescSrc, "VirtIO")) {
        //if (strstr(AdapterDescSrc, "VirtIO")) {
        GatewayIP = pIpAdapterInfo->GatewayList.IpAddress.String;
        return  true;
      }
      pIpAdapterInfo = pIpAdapterInfo->Next;
    }
  }
  //释放内存空间
  if (pIpAdapterInfo) {
    delete[] pIpAdapterInfo;
    pIpAdapterInfo = NULL;
  }
  return false;

#else
  char gateway[32] = { 0 };
  char buffer[200] = { 0 };
  unsigned long bufLen = sizeof(buffer);

  unsigned long defaultRoutePara[4] = { 0 };
  FILE * pfd = fopen("/proc/net/route", "r");
  if (NULL == pfd) return false;

  while (fgets(buffer, bufLen, pfd)) {
    sscanf(buffer, "%*s %x %x %x %*x %*x %*x %x %*x %*x %*x\n", (unsigned int *)&defaultRoutePara[1], (unsigned int *)&defaultRoutePara[0], (unsigned int *)&defaultRoutePara[3], (unsigned int *)&defaultRoutePara[2]);

    if (NULL != strstr(buffer, "VirtIO")) {
      //如果FLAG标志中有 RTF_GATEWAY
      if (defaultRoutePara[3] & RTF_GATEWAY) {
        unsigned long ip = defaultRoutePara[0];
        snprintf(gateway, 32, "%d.%d.%d.%d", (ip & 0xff), (ip >> 8) & 0xff, (ip >> 16) & 0xff, (ip >> 24) & 0xff);
        break;
      }
    }

    memset(buffer, 0, bufLen);
  }

  fclose(pfd);
  pfd = NULL;
  GatewayIP = gateway;
  return true;

#endif // _WIN32	
}
