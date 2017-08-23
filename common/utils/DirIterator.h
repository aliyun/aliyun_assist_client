#pragma once

#include <string>

#if !defined _WIN32
#include <dirent.h>
#else
#include "windows.h"
#endif

/** Simple class for iterating over the files in a directory
  * and reporting their names and types.
  */
class DirIterator {
 public:
  DirIterator(const char* path);
  ~DirIterator();

  // iterate to the next entry in the directory
  bool next();

  // methods to return information about
  // the current entry
  std::string fileName() const;
  std::string filePath() const;
  bool isDir() const;

 private:
  std::string m_path;

#if !defined _WIN32
  DIR* m_dir;
  dirent* m_entry;
#endif

#if defined _WIN32
  HANDLE m_findHandle;
  WIN32_FIND_DATAA _findData;
  bool m_firstEntry;
#endif
};

