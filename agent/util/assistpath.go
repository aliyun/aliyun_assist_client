package util

import (
	"errors"
	"os"
	"strings"
)

func MakeSurePath(path string) {
	os.MkdirAll(path, os.ModePerm)
}

func SetCurrentEnvPath() bool {
	path := os.Getenv("path")
    path += ";"
	cur_path,_ := GetCurrentPath()
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

func GetScriptPath() (string, error) {
	var cur string
	var err error
  	cur, err = GetCurrentPath()

	path := cur + "../work/" + "script"
	// TODO: MakeSurePath would not alwyas succeed, retrieve its error and return
	MakeSurePath(path)
  	return path, err
}

func GetHybridPath() (string, error) {
	var cur string
	var err error
	cur, err = GetCurrentPath()

	path := cur + "../hybrid"
	MakeSurePath(path)
	return path, err
}

func GetTempPath() (string, error) {
	goTempDir := os.TempDir()

	// According to https://pkg.go.dev/os#TempDir, path returned from os.TempDir()
	// is neither guaranteed to exist nor have accessible permissions. Therefore
	// we need to make sure such path accessible manually.
	MakeSurePath(goTempDir)

	return goTempDir, nil
}
