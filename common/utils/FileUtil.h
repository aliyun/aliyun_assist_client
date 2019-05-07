#pragma once

#include <exception>
#include <string>

class FileUtils {
 public:
  static std::string dirname(const char* path);
  static void mkpath(const char* dir);
  static bool fileExists(const char* path);
  static void mkdir(const char* dir);
  static void copyFile(const char* src, const char* dest);
  static char separator();
  static void rmdirRecursive(const char* path);
  static void rmdir(const char* dir);
  static void removeFile(const char* src);
  static bool readFile(const std::string& path, std::string& content);
  static bool writeFile(const std::string& file, const std::string& content);
};

