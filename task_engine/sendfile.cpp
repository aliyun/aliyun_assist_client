// FileOp.cpp: 定义应用程序的入口点。
//

#include "sendfile.h"
#include <string>
#include <fstream>
#ifdef _MSC_VER
#include <io.h>
#else
#include <sys/stat.h>
#include <grp.h>
#include <unistd.h>
#include <pwd.h>
#include <sys/types.h>
#endif
#include "utils/AssistPath.h"
#include "utils/encoder.h"
#include "md5/md5.h"
#include "utils/host_finder.h"
#include "utils/http_request.h"
#include "utils/Log.h"
#include "utils/service_provide.h"
#include "utils/CStringUtil.h"
using namespace std;

enum  eContentType
{
	PlainText,
	Base64,
	Url
};


//enum eFileType
//{
//	Text,
//	Binary
//};

eSendFileStatus write_file(string file_path, const char* data, size_t len, /*eFileType ftype,*/ bool over_write) {

	bool fileExist = (access(file_path.c_str(), 0) == 0);
	//文件存在且不允许覆写，返回失败
	if (fileExist && !over_write) {
		return eSendFileStatus::eFileAlreadyExist;
	}
	std::ofstream ofs;
	//if (ftype == eFileType::Binary) {
		ofs.open(file_path, std::ofstream::out | std::ofstream::binary | std::ofstream::trunc);
	//}
	//else {
	//	ofs.open(file_path, std::ofstream::out | std::ofstream::trunc);
	//}

	if (!ofs.is_open()) {
		return eFileCreateFail;
	}
	ofs.write(data, len);
	return eSendFileStatus::eSuccess;
}

#ifdef _MSC_VER
#else

__uid_t GetUserIdByName(const std::string& UserName) {
	struct passwd *user = getpwnam(UserName.c_str());
	if (NULL == user) {
		return (__uid_t)(-1);
	}
	return user->pw_uid;
}

__gid_t GetGroupIdByName(const std::string& GroupName) {
	struct group* data = getgrnam(GroupName.c_str());
	if (NULL == data) {
		return (__gid_t)(-1);
	}
	return data->gr_gid;
}

eSendFileStatus SetFileAttribute(const std::string& path, uid_t owner, gid_t group, const std::string& mode) {
	std::string tmpMode;
	
	if (mode.length() != 3 && mode.length() != 4 && mode.length() != 0) {
		return eSendFileStatus::eInalidFileMode;
	}
	if (mode.length() == 3) {
		tmpMode.append("0");
	}
	if (mode.length() == 0) {
		tmpMode.append("0644");
	}
	tmpMode.append(mode);
	long intMode = strtol(tmpMode.c_str(), NULL, 8);
	if (-1 == intMode) {
		return eSendFileStatus::eInalidFileMode;
	}
	int ret = chown(path.c_str(), owner, group);
	if (ret != 0) {
		return eSendFileStatus::eChownError;
	}
	ret = chmod(path.c_str(), intMode);
	if (ret != 0) {
		return eSendFileStatus::eChmodError;
	}
	return eSendFileStatus::eSuccess;
}
#endif

eSendFileStatus SendFileImp(const task_engine::SendFile& sendFile) {
	string file_path;
	if (sendFile.name.empty()) {
		return eSendFileStatus::eInvalidFilePath;
	}
	if (sendFile.content.empty()) {
		return eSendFileStatus::eEmptyContent;
	}
	
	if (sendFile.destination.empty()) {
#ifdef _MSC_VER
		//windows版默认为云助手目录
		AssistPath assistPath("");
		std::string dir = assistPath.GetCurrDir();
		file_path = dir + "\\" + sendFile.name;
#else
		//linux版默认为root目录
		file_path = "/root/" + sendFile.name;
#endif
	}
	else {
		file_path = sendFile.destination;
#ifdef _MSC_VER
		if (file_path.c_str()[file_path.length() - 1] != '\\') {
			file_path.append("\\");
		}
#else
		if (file_path.c_str()[file_path.length() - 1] != '/') {
			file_path.append("/");
		}	
#endif
		file_path.append(sendFile.name);
	}
	if (!sendFile.destination.empty()) {
		AssistPath assistPath("");
		int create_dir = assistPath.CreateDirRecursive(file_path);
		if (0 != create_dir) {
			return eSendFileStatus::eCreateDirFailed;
		}
	}
	string fileContent;
	Encoder encode;
	size_t len = 0;
	unsigned char* p = encode.B64DecodeEx(sendFile.content.c_str(), sendFile.content.size(), &len);
	if (NULL == p || 0 == len) {
		return eSendFileStatus::eInvalidContent;
	}
	fileContent.append((char*)p, len);

	std::string content_md5 = md5(sendFile.content);
	CStringUtil::ToLower(content_md5);
	string tmpMd5 = sendFile.signature;
	CStringUtil::ToLower(tmpMd5);
	if (content_md5 != tmpMd5) {
		return eSendFileStatus::eInvalidSignature;
	}

#ifdef _MSC_VER
#else
	__gid_t gid = 0;
	if (!sendFile.group.empty()) {
		gid = GetGroupIdByName(sendFile.group);
	}
	
	if (gid == (__gid_t)(-1)) {
		return eSendFileStatus::eInalidGID;
	}
	__uid_t uid = 0;
	if (!sendFile.owner.empty()) {
		uid = GetUserIdByName(sendFile.owner);
	}
	if (uid == (__uid_t)(-1)) {
		return eSendFileStatus::eInalidUID;
	}
#endif

	//eFileType fType = eFileType::Binary;
	//if (sendFile.fileType.empty() || sendFile.fileType == "Text") {
	//	fType = eFileType::Text;
	//}
	//else if (sendFile.fileType == "Binary") {
	//	fType = eFileType::Binary;
	//}
	//else {
	//	return eSendFileStatus::eInvalidFileType;
	//}

	eSendFileStatus ret = write_file(file_path, fileContent.c_str(), fileContent.length(), /*fType,*/ sendFile.overwrite);
	if (ret != eSendFileStatus::eSuccess) {
		return ret;
	}
#ifdef _MSC_VER
	return eSendFileStatus::eSuccess;
#else
	if (sendFile.owner.empty() && sendFile.group.empty() && sendFile.mode.empty()) {
		return eSendFileStatus::eSuccess;
	}
	return SetFileAttribute(file_path, uid, gid, sendFile.mode);
#endif
}

void SendFileFinished(const task_engine::SendFile& sendFile, eSendFileStatus status) {
	if (HostFinder::getServerHost().empty()) {
		Log::Error("Get server host failed");
		return;
	}
	std::string url = ServiceProvide::GetFinishOutputService();
	char param[512];
	string reportStatus = "Success";
	if (status != eSendFileStatus::eSuccess) {
		reportStatus = "Failed";
	}
	sprintf(param, "?taskId=%s&status=%s&taskType=%s&errorcode=%d",
		sendFile.invokeId.c_str(), reportStatus.c_str(), "sendfile",status);
	url += param;
	Log::Info("post = %s", url.c_str());
	std::string response;
	bool ret = HttpRequest::https_request_post_text(url, "", response);
	if (!ret) {
		Log::Error("SendFileFinished post error %s %s", sendFile.invokeId.c_str(), response.c_str());
	}
	Log::Info("SendFileFinished post ok %s %s", sendFile.invokeId.c_str(), response.c_str());
}

void SendFileInvalid(const task_engine::SendFile& sendFile, eSendFileStatus status) {
	if (HostFinder::getServerHost().empty()) {
		Log::Error("Get server host failed");
		return;
	}
	std::string url = ServiceProvide::GetInvalidTaskService();
	char param[512];
	std::string key;
	std::string value;
	if (status == eSendFileStatus::eInvalidFilePath) {
		key = "FileNameInvalid";
		value = sendFile.name;
	}else if (status == eSendFileStatus::eFileAlreadyExist) {
		key = "FileExist";
		value = sendFile.name;
	}
	else if (status == eSendFileStatus::eEmptyContent) {
		key = "EmptyFile";
	}
	else if (status == eSendFileStatus::eInvalidContent) {
		key = "InvalidFileContent";
	}
	//else if (status == eSendFileStatus::eInvalidContentType) {
	//	key = "InvalidContentType";
	//	value = sendFile.contentType;
	//}
	else if (status == eSendFileStatus::eInvalidFileType) {
		key = "InvalidFileType";
		value = sendFile.fileType;
	}
	else if (status == eSendFileStatus::eInvalidSignature) {
		key = "InvalidSignature";
		value = sendFile.signature;
	}
	else if (status == eSendFileStatus::eInalidFileMode) {
		key = "InvalidFileMode";
		value = sendFile.mode;
	}
	else if (status == eSendFileStatus::eInalidGID) {
		key = "FileGroupNotExist";
		value = sendFile.group;
	}
	else if (status == eSendFileStatus::eInalidUID) {
		key = "FileOwnerNotExist";
		value = sendFile.owner;
	}
	url = url + "?" + "taskId=" + sendFile.invokeId + "&taskType=sendfile&param=" + key + "&value=" + value;
	Log::Info("post = %s", url.c_str());
	std::string response;
	bool ret = HttpRequest::https_request_post_text(url, "", response);
	if (!ret) {
		Log::Error("SendFileInvalid post error %s %s", sendFile.invokeId.c_str(), response.c_str());
	}
	Log::Info("SendFileInvalid post ok %s %s", sendFile.invokeId.c_str(), response.c_str());
}

bool doSendFile(const task_engine::SendFile& sendFile) {
	eSendFileStatus ret = SendFileImp(sendFile);
	Log::Info("sendfile ret %d", ret);
	if (ret <= eCreateDirFailed) {
		SendFileFinished(sendFile, ret);
	}
	else {
		SendFileInvalid(sendFile, ret);
	}
	if (eSendFileStatus::eSuccess == ret) {
		return true;
	}
	return false;
}

//int test()
//{
//	unlink("c:\\aliwork\\yyy.txt");
//	unlink("c:\\aliwork\\yyyq.txt");
//	if (write_file("c:\\aliwork\\yyy.txt", "youyong", 7, eFileType::Text, false) == eSendFileStatus::eSuccess) {
//		cout << "ok" << endl;
//	}
//	else {
//		cout << "error" << endl;
//	}
//	if (write_file("c:\\aliwork\\yyy.txt", "youyong", 7, eFileType::Text, false) == eSendFileStatus::eFileAlreadyExist) {
//		cout << "ok" << endl;
//	}
//	else {
//		cout << "error" << endl;
//	}
//	if (write_file("c:\\aliwork\\yyy.txt", "youyong", 7, eFileType::Text, true) == eSendFileStatus::eSuccess) {
//		cout << "ok" << endl;
//	}
//	else {
//		cout << "error" << endl;
//	}
//	if (write_file("c:\\aliwork\\yyyq.txt", "youyong", 7, eFileType::Binary, false) == eSendFileStatus::eSuccess) {
//		cout << "ok" << endl;
//	}
//	else {
//		cout << "error" << endl;
//	}
//	if (write_file("c:\\aliwork\\yyyq.txt", "youyong", 7, eFileType::Binary, false) == eSendFileStatus::eFileAlreadyExist) {
//		cout << "ok" << endl;
//	}
//	else {
//		cout << "error" << endl;
//	}
//	if (write_file("c:\\aliwork\\yyyq.txt", "youyong", 7, eFileType::Binary, true) == eSendFileStatus::eSuccess) {
//		cout << "ok" << endl;
//	}
//	else {
//		cout << "error" << endl;
//	}
//	return 0;
//}
