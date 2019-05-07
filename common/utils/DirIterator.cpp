#include "DirIterator.h"

#if !defined _WIN32
#include <dirent.h>
#endif

#include <string.h>

namespace {

inline bool endsWith(const std::string& str, const char* text) {
  size_t length = strlen(text);
  return str.find(text, str.size() - length) != std::string::npos;
}

}

DirIterator::DirIterator(const char* path) {
  m_path = path;

#if !defined _WIN32
  m_dir = opendir(path);
  m_entry = 0;
#else
  // to list the contents of a directory, the first
  // argument to FindFirstFile needs to be a wildcard
  // of the form: C:\path\to\dir\*
  std::string searchPath = m_path;
  if (!endsWith(searchPath,"/")) {
    searchPath.append("/");
  }
  searchPath.append("*");
  m_findHandle = FindFirstFileA(searchPath.c_str(),&_findData);
  m_firstEntry = true;
#endif
}

DirIterator::~DirIterator() {
#if !defined _WIN32
  closedir(m_dir);
#else
  FindClose(m_findHandle);
#endif
}

bool DirIterator::next() {
#if !defined _WIN32
  m_entry = readdir(m_dir);
  return m_entry != 0;
#else
  bool result;
  if (m_firstEntry) {
    m_firstEntry = false;
    return m_findHandle != INVALID_HANDLE_VALUE;
  } else {
    result = FindNextFileA(m_findHandle,&_findData);
  }
  return result;
#endif
}

std::string DirIterator::fileName() const {
#if !defined _WIN32
  return m_entry->d_name;
#else
  return _findData.cFileName;
#endif
}

std::string DirIterator::filePath() const {
  return m_path + '/' + fileName();
}

bool DirIterator::isDir() const {
#if !defined _WIN32
  return m_entry->d_type == DT_DIR;
#else
  return (_findData.dwFileAttributes & FILE_ATTRIBUTE_DIRECTORY) != 0;
#endif
}

