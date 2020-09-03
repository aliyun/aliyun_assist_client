/**************************************************************************
Copyright: ALI
Author: tmy
Date: 2017-03-14
Type: .cpp
Description: Provide functions to resolve server address and check net
**************************************************************************/
#include "host_finder.h"
#include <algorithm>
#include <mutex>
#include <vector>

#include "utils/MutexLocker.h"
#include "utils/http_request.h"
#include "Log.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"

std::vector<std::string> region_ids = { "cn-qingdao",
"cn-beijing",
"cn-zhangjiakou",
"cn-huhehaote",
"cn-hangzhou",
"cn-shanghai",
"cn-shenzhen",
"cn-chengdu",
"cn-hongkong",
"ap-southeast-1",
"ap-southeast-2",
"ap-southeast-3",
"ap-southeast-5",
"ap-south-1",
"ap-northeast-1",
"us-west-1",
"us-east-1",
"eu-central-1",
"eu-west-1",
"me-east-1",
"cn-north-2-gov-1",
"cn-qingdao-nebula"};

bool HostFinder::stopPolling = false;
bool HostFinder::connectionDetect(string regionId) {
	std::string host = regionId + ".axt.aliyun.com";
	string url = "https://" + host + "/luban/api/connection_detect";

	string response;
	bool status = HttpRequest::https_request_get(url.c_str(), response);
	if (status) {
		return true;
	}
	return false;
};

// works only on classic network
bool HostFinder::requestRegionId(string regionId) {
  std::string host = regionId + ".axt.aliyun.com";
  string url = "https://" + host + "/luban/api/classic/region-id";

  string response;
  bool status = HttpRequest::https_request_get(url.c_str(), response);
  if (status) {
    if (response.find(regionId) != string::npos) {
      return true;
    }
    return false;
  }

  return false;
};

string HostFinder::getRegionIdInFile() {
	AssistPath  path_service;
	std::string cur_dir = path_service.GetCurrDir();
	std::string region_file = cur_dir + FileUtils::separator() + ".." + FileUtils::separator() + "region-id";
	if ( !FileUtils::fileExists(region_file.c_str()) ){
		return "";
	}

	std::string regionId;
	FileUtils::readFile(region_file, regionId);
	regionId.erase( 0, regionId.find_first_not_of(" \n\r\t") );
	regionId.erase(regionId.find_last_not_of(" \n\r\t") + 1);

	if (requestRegionId(regionId) ) {
		return regionId;
	}
	return "";
};



string HostFinder::getRegionIdInVpc() {
	string regionId;
	string url  = "http://100.100.100.200/latest/meta-data/region-id";
	bool status = HttpRequest::http_request_get(url.c_str(), regionId);
	if ( !status ){
		return "";
	}
	
	if ( connectionDetect(regionId) ) {
		return regionId;
	}
	return "";
};

string HostFinder::getRegionId()  {
	static std::string regionId;
	static std::mutex  mutex;

	MutexLocker(&mutex) {
		if (regionId.size()) {
			return regionId;
		}

    string result = getRegionIdInVpc();
		if (result.size()) {
			regionId = result;
			return result;
		}

		result = getRegionIdInFile();
		if (result.size()) {
			regionId = result;
			return result;
		}

    result = pollingRegionId();
    if (result.size()) {
      regionId = result;
      return result;
    }
	}
	return "";
}

void HostFinder::setStopPolling(bool flag){
	stopPolling = flag;
}

string HostFinder::pollingRegionId() {
  auto iter = region_ids.begin();
  for (; iter != region_ids.end(); ++iter) {
	if (stopPolling) {
		break;
	}
    if (requestRegionId(*iter)) {
      return *iter;
    }
  }

  return "";
}

string HostFinder::getServerHost() {
	string regionId = getRegionId();
	if ( regionId.size() ) {
		return regionId + ".axt.aliyun.com";
	}
	else {
		return "";
	}
};


