/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-14
Type: .h
Description: Provide functions to resolve server address and check net
**************************************************************************/

#ifndef PROJECT_CHECKNET_H_
#define PROJECT_CHECKNET_H_

#include <string.h>
#include <stdio.h>
#include <iostream>

using  std::string;

class HostChooser {
 public:
  static string m_HostSelect;
  static bool m_Classical;
  bool Init(string path);

 private:
  //bool CheckOutNet(string ip);
  bool WaitNetworkReady(int timeout);

  bool FindServer(string file, string & server);
  bool ChooseAddress(string config, string &ip);
  bool FileRead(string file, string & content);
  bool FileWrite(string file, string content);

  bool GetAdaptInfo(string & GatewayIP);
};

#endif //PROJECT_CHECKNET_H_


