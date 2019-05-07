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

#include "utils/MutexLocker.h"
#include "utils/http_request.h"
#include "Log.h"
#include "utils/AssistPath.h"
#include "utils/FileUtil.h"




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

	if ( connectionDetect(regionId) ) {
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


