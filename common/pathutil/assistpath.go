package pathutil

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	scriptPath string

	logPath       string
	logPathRWLock sync.RWMutex

	versionedConfigDir       string
	versionedConfigDirRWLock sync.RWMutex

	crossVersionConfigDir       string
	crossVersionConfigDirRWLock sync.RWMutex
)

func MakeSurePath(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func SetCurrentEnvPath() bool {
	path := os.Getenv("path")
	path += ";"
	cur_path, _ := GetExecutableDir()
	path += cur_path
	os.Setenv("path", path)
	return true
}

func SetScriptPath(path string) {
	scriptPath = path
}

func GetScriptPath() (string, error) {
	if scriptPath != "" {
		return scriptPath, nil
	}

	crossVersionDir, err := getCrossVersionOutboundDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(crossVersionDir, "work", "script")
	err = MakeSurePath(path)
	return path, err
}

func SetLogPath(path string) {
	logPath = path
	MakeSurePath(logPath)
}

func GetLogPath() (string, error) {
	logPathRWLock.RLock()
	if logPath == "" {
		logPathRWLock.RUnlock()
		logPathRWLock.Lock()
		defer logPathRWLock.Unlock()

		if logPath == "" {
			parentDir, err := GetExecutableDir()
			if err != nil {
				return "", err
			}

			writableDir := filepath.Join(parentDir, "log")
			if err := MakeSurePath(writableDir); err != nil {
				return "", err
			}

			logPath = writableDir
		}
	} else {
		defer logPathRWLock.RUnlock()
	}

	return logPath, nil
}

func GetHybridPath() (string, error) {
	crossVersionDir, err := GetCrossVersionInboundDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(crossVersionDir, "hybrid")
	err = MakeSurePath(path)
	return path, err
}

func GetConfigPath() (string, error) {
	versionedConfigDirRWLock.RLock()
	if versionedConfigDir == "" {
		versionedConfigDirRWLock.RUnlock()
		versionedConfigDirRWLock.Lock()
		defer versionedConfigDirRWLock.Unlock()

		if versionedConfigDir == "" {
			parentDir, err := getVersionedInboundDir()
			if err != nil {
				return "", err
			}

			writableDir := filepath.Join(parentDir, "config")
			if err := MakeSurePath(writableDir); err != nil {
				return "", err
			}

			versionedConfigDir = writableDir
		}
	} else {
		defer versionedConfigDirRWLock.RUnlock()
	}

	return versionedConfigDir, nil
}

func GetCrossVersionConfigPath() (string, error) {
	crossVersionConfigDirRWLock.RLock()
	if crossVersionConfigDir == "" {
		crossVersionConfigDirRWLock.RUnlock()
		crossVersionConfigDirRWLock.Lock()
		defer crossVersionConfigDirRWLock.Unlock()

		if crossVersionConfigDir == "" {
			crossVersionDir, err := GetCrossVersionInboundDir()
			if err != nil {
				return "", err
			}

			writableDir := filepath.Join(crossVersionDir, "config")
			if err := MakeSurePath(writableDir); err != nil {
				return "", err
			}

			crossVersionConfigDir = writableDir
		}
	} else {
		defer crossVersionConfigDirRWLock.RUnlock()
	}

	return crossVersionConfigDir, nil
}

func GetTempPath() (string, error) {
	goTempDir := os.TempDir()

	// According to https://pkg.go.dev/os#TempDir, path returned from os.TempDir()
	// is neither guaranteed to exist nor have accessible permissions. Therefore
	// we need to make sure such path accessible manually.
	err := MakeSurePath(goTempDir)

	return goTempDir, err
}

func GetCachePath() (string, error) {
	crossVersionDir, err := getCrossVersionOutboundDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(crossVersionDir, "cache")
	MakeSurePath((path))
	return path, err
}

func GetPluginPath() (string, error) {
	crossVersionDir, err := getCrossVersionOutboundDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(crossVersionDir, "plugin")
	err = MakeSurePath(path)
	return path, err
}

func GetPreInstalledPluginPath() (string, error) {
	cur, err := getVersionedOutboundDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(cur, "plugin")
	err = MakeSurePath(path)
	return path, err
}
