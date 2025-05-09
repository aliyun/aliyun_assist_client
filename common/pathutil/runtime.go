package pathutil

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	_currentVersionDir       string
	_currentVersionDirRWLock sync.RWMutex

	_crossVersionDir       string
	_crossVersionDirRWLock sync.RWMutex
)

func getCurrentVersionDir() (string, error) {
	_currentVersionDirRWLock.RLock()
	if _currentVersionDir == "" {
		_currentVersionDirRWLock.RUnlock()
		_currentVersionDirRWLock.Lock()
		defer _currentVersionDirRWLock.Unlock()

		if _currentVersionDir == "" {
			path, err := os.Executable()
			if err != nil {
				return "", err
			}

			_currentVersionDir = filepath.Dir(path)
		}
	} else {
		defer _currentVersionDirRWLock.RUnlock()
	}

	return _currentVersionDir, nil
}

func GetCrossVersionDir() (string, error) {
	_crossVersionDirRWLock.RLock()
	if _crossVersionDir == "" {
		_crossVersionDirRWLock.RUnlock()
		_crossVersionDirRWLock.Lock()
		defer _crossVersionDirRWLock.Unlock()

		if _crossVersionDir == "" {
			versionDir, err := getCurrentVersionDir()
			if err != nil {
				return "", err
			}

			absoluteVersionDir, err := filepath.Abs(versionDir)
			if err != nil {
				return "", err
			}
			// Although filepath.Dir method would call filepath.Clean internally, here
			// explicitly call the method to guarantee no trailing slash in path
			cleanedVersionDir := filepath.Clean(absoluteVersionDir)

			_crossVersionDir = filepath.Dir(cleanedVersionDir)
		}
	} else {
		defer _crossVersionDirRWLock.RUnlock()
	}

	return _crossVersionDir, nil
}
