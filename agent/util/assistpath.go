package util

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var scriptPath = ""

func MakeSurePath(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func SetCurrentEnvPath() bool {
	path := os.Getenv("path")
	path += ";"
	cur_path, _ := GetCurrentPath()
	path += cur_path
	os.Setenv("path", path)
	return true
}

func GetCurrentPath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	i := strings.LastIndex(path, "/")
	if i < 0 {
		i = strings.LastIndex(path, "\\")
	}
	if i < 0 {
		return "", errors.New(`error: Can't find "/" or "\".`)
	}
	return string(path[0 : i+1]), nil
}

func SetScriptPath(path string) {
	scriptPath = path
}

func GetScriptPath() (string, error) {
	if scriptPath != "" {
		return scriptPath, nil
	}
	var cur string
	var err error
	cur, err = GetCurrentPath()

	path := cur + "../work/" + "script"
	err = MakeSurePath(path)
	return path, err
}

func GetHybridPath() (string, error) {
	var cur string
	var err error
	cur, err = GetCurrentPath()

	path := cur + "../hybrid"
	err = MakeSurePath(path)
	return path, err
}

func GetConfigPath() (string, error) {
	currentVersionDir, err := GetCurrentPath()
	if err != nil {
		return "", err
	}

	currentVersionConfigDir := filepath.Join(currentVersionDir, "config")
	if err := MakeSurePath(currentVersionConfigDir); err != nil {
		return "", err
	}

	return currentVersionConfigDir, nil
}

func GetCrossVersionConfigPath() (string, error) {
	crossVersionDir, err := getCrossVersionDir()
	if err != nil {
		return "", err
	}

	crossVersionConfigDir := filepath.Join(crossVersionDir, "config")
	if err := MakeSurePath(crossVersionConfigDir); err != nil {
		return "", err
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

func getCrossVersionDir() (string, error) {
	currentVersionDir, err := GetCurrentPath()
	if err != nil {
		return "", err
	}

	absoluteCurrentVersionDir, err := filepath.Abs(currentVersionDir)
	if err != nil {
		return "", err
	}
	// Although filepath.Dir method would call filepath.Clean internally, here
	// explicitly call the method to guarantee no trailing slash in path
	cleanedCurrentVersionDir := filepath.Clean(absoluteCurrentVersionDir)

	multiVersionDir := filepath.Dir(cleanedCurrentVersionDir)
	return multiVersionDir, nil
}

func GetCachePath() (string, error) {
	cur, err := GetCurrentPath()
	if err != nil {
		return "", err
	}

	path := filepath.Join(cur, "..", "cache")
	MakeSurePath((path))
	return path, err
}

func GetPluginPath() (string , error) {
	cur, err := GetCurrentPath()
	if err != nil {
		return "", err
	}
	// Use filepath.Join() to clean the parent directory element `..` in path
	path := filepath.Join(cur, "..", "plugin")
	err = MakeSurePath(path)
	return path, err
}
