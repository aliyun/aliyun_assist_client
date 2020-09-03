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
class HostFinder {
 public:
  static string getRegionId();
  static string getServerHost();
  static void setStopPolling(bool flag);
 private:
  static string getRegionIdInVpc();
  static string getRegionIdInFile();
  static string pollingRegionId();
  static bool connectionDetect(string regionId);

  // works only on classic network
  static bool requestRegionId(string regionId);
  static bool stopPolling;
};

#endif //PROJECT_CHECKNET_H_


