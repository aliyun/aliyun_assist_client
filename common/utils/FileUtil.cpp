#include "FileUtil.h"

#include "utils/Log.h"
#include "utils/DirIterator.h"

#include <algorithm>
#include <assert.h>
#include <string.h>
#include <fstream>
#include <sstream>
#include <iostream>

#if defined(_WIN32)
#include "windows.h"
#else
#include <unistd.h>
#include <stdio.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/time.h>
#include <sys/stat.h>
#include <errno.h>
#include <libgen.h>
#endif

std::string FileUtils::dirname(const char* path) {
#if !defined _WIN32
  char* pathCopy = strdup(path);
  std::string dirname = ::dirname(pathCopy);
  free(pathCopy);
  return dirname;
#else
  char drive[3];
  char dir[MAX_PATH];

  _splitpath_s(path,
               drive, /* drive */
               3, /* drive length */
               dir,
               MAX_PATH, /* dir length */
               0, /* filename */
               0, /* filename length */
               0, /* extension */
               0  /* extension length */
              );

  std::string result;
  if (drive[0]) {
    result += std::string(drive);
  }
  result += dir;

  return result;
#endif
}

char FileUtils::separator() {
#ifdef _WIN32
  return '\\';
#else
  return '/';
#endif
}

void FileUtils::mkpath(const char* dir) {
  std::string currentPath;
  std::istringstream stream(dir);
  while (!stream.eof()) {
    std::string segment;
    std::getline(stream, segment, '/');
    currentPath += segment;
    if (!currentPath.empty() && !fileExists(currentPath.c_str())) {
      mkdir(currentPath.c_str());
    }
    currentPath += '/';
  }
}


bool FileUtils::readFile(const std::string& file, std::string &content) {

	Log::Info("Read file :" + file);
	FILE *fp = fopen(file.c_str(), "r");
	if (!fp) {
		Log::Error("File open error!");
		return false;
	}

	char *pBuf = nullptr;
	fseek(fp, 0, SEEK_END);
	long len = ftell(fp); //获取文件长度
	if (len < 0) {
		fclose(fp);
		return false;
	}
	pBuf = new char[len + 1];
	memset(pBuf, 0, sizeof(char) * (len + 1));

	fseek(fp, 0, SEEK_SET);
	int count = fread(pBuf, sizeof(char), len, fp);

	pBuf[len] = 0;
	fclose(fp);

	content = pBuf;
	delete[] pBuf;
	return true;
}

bool FileUtils::writeFile(const std::string& file, const std::string& content) {

	Log::Info("Write file :" + file);
	FILE *fp = fopen(file.c_str(), "w");;
	if (!fp) {
		Log::Error("File open error!");
		return false;
	}

	fwrite(content.c_str(), 1, content.length(), fp);
	fflush(fp);
	fclose(fp);
	return true;
}




void FileUtils::rmdirRecursive(const char* path) {
  // remove dir contents
  DirIterator dir(path);
  while (dir.next()) {
    std::string name = dir.fileName();
    if (name != "." && name != "..") {
      if (dir.isDir()) {
        rmdirRecursive(dir.filePath().c_str());
      } else {
        removeFile(dir.filePath().c_str());
      }
    }
  }

  // remove the directory itself
  rmdir(path);
}

void FileUtils::rmdir(const char* dir) {
#if !defined _WIN32
  ::rmdir(dir);

#else
  RemoveDirectoryA(dir);
#endif
}

void FileUtils::removeFile(const char* src) {
#if !defined _WIN32
  unlink(src);
#else
  DeleteFileA(src);
#endif
}

bool FileUtils::fileExists(const char* path) {
#if !defined _WIN32
  struct stat fileInfo;
  if (lstat(path, &fileInfo) != 0) {
    return false;
  }
  return true;
#else
  DWORD result = GetFileAttributesA(path);
  if (result == INVALID_FILE_ATTRIBUTES) {
    return false;
  }
  return true;
#endif
}

void FileUtils::mkdir(const char* dir) {
#if !defined _WIN32
  if (::mkdir(dir, S_IRWXU | S_IRGRP | S_IXGRP | S_IROTH | S_IXOTH) != 0) {
    return;
  }
#else
  if (!CreateDirectoryA(dir, 0 /* default security attributes */)) {
    return;
  }
#endif
}

void FileUtils::copyFile(const char* src, const char* dest) {
#if !defined _WIN32
  std::ifstream inputFile(src, std::ios::binary);
  std::ofstream outputFile(dest, std::ios::binary | std::ios::trunc);

  if (!inputFile.good()) {
    Log::Error("copy file failed");
  }
  if (!outputFile.good()) {
    Log::Error("copy file failed");
  }

  outputFile << inputFile.rdbuf();

  if (inputFile.bad()) {
    Log::Error("copy file failed");
  }
  if (outputFile.bad()) {
    Log::Error("copy file failed");
  }
#else
  if (!CopyFileA(src, dest, FALSE)) {
    Log::Error("copy file failed");
  }
#endif
}

