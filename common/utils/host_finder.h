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

 private:
  static string getRegionIdInVpc();
  static string getRegionIdInFile();
  static bool   connectionDetect(string regionId);
};

#endif //PROJECT_CHECKNET_H_


